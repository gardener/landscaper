// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	lserrors "github.com/gardener/landscaper/apis/errors"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"k8s.io/apimachinery/pkg/selection"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	validation "github.com/gardener/landscaper/apis/core/validation"
)

// DeployerClusterRoleName is the name of the deployer cluster role.
// That role has access to the deployer needed artifacts like deploy items and secrets.
const DeployerClusterRoleName = "landscaper:deployer"

type DeployerManagement struct {
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme
	config config.DeployerManagementConfiguration
}

// NewDeployerManagement creates a new deployer manager.
func NewDeployerManagement(log logr.Logger, client client.Client, scheme *runtime.Scheme, config config.DeployerManagementConfiguration) *DeployerManagement {
	return &DeployerManagement{
		log:    log,
		client: client,
		scheme: scheme,
		config: config,
	}
}

// Reconcile reconciles a deployer installation given a deployer registration and a environment.
func (dm *DeployerManagement) Reconcile(ctx context.Context, registration *lsv1alpha1.DeployerRegistration, env *lsv1alpha1.Environment) error {
	registration = registration.DeepCopy()
	env = env.DeepCopy()

	inst, err := dm.getInstallation(ctx, registration, env)
	if err != nil {
		return err
	}

	envTargetSelectors := []lsv1alpha1.TargetSelector{}
	for _, selector := range env.Spec.TargetSelectors {
		if registration.Name == "helm" {
			selector.Annotations = append(
				selector.Annotations,
				lsv1alpha1.Requirement{
					Key:      lsv1alpha1.DeployerOnlyTargetAnnotationName,
					Operator: selection.DoesNotExist,
				},
			)
		}

		envTargetSelectors = append(envTargetSelectors, selector)
	}

	targetSelectorsBytes, err := json.Marshal(envTargetSelectors)
	if err != nil {
		return fmt.Errorf("unable to marshal target selectors: %w", err)
	}

	_, err = dm.Writer().CreateOrUpdateCoreInstallation(ctx, read_write_layer.W000002, inst, func() error {
		controllerutil.AddFinalizer(inst, lsv1alpha1.LandscaperDMFinalizer)
		inst.Spec.ComponentDescriptor = registration.Spec.InstallationTemplate.ComponentDescriptor
		inst.Spec.Blueprint = registration.Spec.InstallationTemplate.Blueprint
		inst.Spec.Imports = registration.Spec.InstallationTemplate.Imports
		inst.Spec.ImportDataMappings = registration.Spec.InstallationTemplate.ImportDataMappings

		inst.Spec.Imports.Targets = append(inst.Spec.Imports.Targets,
			lsv1alpha1.TargetImport{
				Name:   "cluster",
				Target: "#" + env.Name,
			},
			lsv1alpha1.TargetImport{
				Name:   "landscaperCluster",
				Target: "#" + FQName(registration, env),
			},
		)
		if inst.Spec.ImportDataMappings == nil {
			inst.Spec.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{}
		}
		inst.Spec.ImportDataMappings["releaseName"] = lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", FQName(registration, env))))
		inst.Spec.ImportDataMappings["releaseNamespace"] = lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", env.Spec.Namespace)))
		inst.Spec.ImportDataMappings["identity"] = lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", FQName(registration, env))))
		inst.Spec.ImportDataMappings["targetSelectors"] = lsv1alpha1.NewAnyJSON(targetSelectorsBytes)

		return nil
	})
	if err != nil {
		return err
	}

	if err := dm.createDeployerTarget(ctx, inst, registration, env); err != nil {
		return err
	}

	return err
}

// getInstallation returns the installation for a registration and environment
func (dm *DeployerManagement) getInstallation(ctx context.Context,
	registration *lsv1alpha1.DeployerRegistration,
	env *lsv1alpha1.Environment) (*lsv1alpha1.Installation, error) {
	installations := &lsv1alpha1.InstallationList{}
	if err := read_write_layer.ListInstallations(ctx, dm.client, installations,
		client.InNamespace(dm.config.Namespace),
		client.MatchingLabels{
			lsv1alpha1.DeployerEnvironmentLabelName:  env.Name,
			lsv1alpha1.DeployerRegistrationLabelName: registration.Name,
		},
	); err != nil {
		return nil, fmt.Errorf("unable to list installtions: %w", err)
	}
	if len(installations.Items) == 0 {
		inst := &lsv1alpha1.Installation{}
		inst.Name = FQName(registration, env)

		if len(inst.Name) > validation.InstallationNameMaxLength {
			err := lserrors.NewError(
				"getInstallation",
				"installation name max length exceeded",
				fmt.Sprintf("installation name %q in namespace %q exceeds maximum length of %d (environment %q)", inst.Name, dm.config.Namespace, validation.InstallationNameMaxLength, env.Name))
			registration.Status.LastError = lserrors.TryUpdateError(inst.Status.LastError, err)

			if err := dm.client.Status().Update(ctx, registration); err != nil {
				dm.log.Error(err, "failed to update status for deployer registration", registration.Name)
			}

			return nil, err
		}

		inst.Namespace = dm.config.Namespace
		inst.Labels = map[string]string{
			lsv1alpha1.DeployerEnvironmentLabelName:  env.Name,
			lsv1alpha1.DeployerRegistrationLabelName: registration.Name,
		}
		return inst, nil
	}
	if len(installations.Items) > 1 {
		return nil, errors.New("more than one installation for the deployer registration and environment found")
	}
	return &installations.Items[0], nil
}

// createDeployerTarget creates a deployer service account and rbac roles
// for a deployer to access deployitems on the landscaper cluster.
func (dm *DeployerManagement) createDeployerTarget(ctx context.Context,
	inst *lsv1alpha1.Installation,
	registration *lsv1alpha1.DeployerRegistration,
	env *lsv1alpha1.Environment) error {
	target := &lsv1alpha1.Target{}
	target.Name = FQName(registration, env)
	target.Namespace = dm.config.Namespace

	// create a new deployer user
	sa := &corev1.ServiceAccount{}
	sa.Name = FQName(registration, env)
	sa.Namespace = dm.config.Namespace

	if _, err := controllerutil.CreateOrUpdate(ctx, dm.client, sa, func() error {
		return controllerutil.SetControllerReference(inst, sa, dm.scheme)
	}); err != nil {
		return err
	}

	restConfig := &rest.Config{}
	restConfig.Host = env.Spec.LandscaperClusterRestConfig.Host
	restConfig.APIPath = env.Spec.LandscaperClusterRestConfig.APIPath
	restConfig.TLSClientConfig.Insecure = env.Spec.LandscaperClusterRestConfig.TLSClientConfig.Insecure
	restConfig.TLSClientConfig.ServerName = env.Spec.LandscaperClusterRestConfig.TLSClientConfig.ServerName
	restConfig.TLSClientConfig.CAData = env.Spec.LandscaperClusterRestConfig.TLSClientConfig.CAData

	if err := kutil.AddServiceAccountAuth(ctx, dm.client, sa, restConfig); err != nil {
		return err
	}

	if err := dm.EnsureRBACRoles(ctx); err != nil {
		return err
	}

	crb := &rbacv1.ClusterRoleBinding{}
	crb.Name = FQName(registration, env)
	if _, err := controllerutil.CreateOrUpdate(ctx, dm.client, crb, func() error {
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     DeployerClusterRoleName,
		}
		crb.Subjects = []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to create cluster role binding for %q: %w", sa.Name, err)
	}

	if _, err := dm.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000074, target, func() error {
		if err := lsutils.BuildKubernetesTarget(target, restConfig); err != nil {
			return err
		}
		// set installation as owner
		return controllerutil.SetControllerReference(inst, target, dm.scheme)
	}); err != nil {
		return err
	}
	return nil
}

// CleanupInstallation cleans up all resources for a deployer installation.
func (dm *DeployerManagement) CleanupInstallation(ctx context.Context, inst *lsv1alpha1.Installation) error {
	envName, ok := inst.Labels[lsv1alpha1.DeployerEnvironmentLabelName]
	if !ok {
		return errors.New("no environment label provided")
	}
	regName, ok := inst.Labels[lsv1alpha1.DeployerRegistrationLabelName]
	if !ok {
		return errors.New("no deployer registration label provided")
	}

	crb := &rbacv1.ClusterRoleBinding{}
	crb.Name = FQNameFromName(regName, envName)
	if err := dm.client.Delete(ctx, crb); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("unable to delete clusterrolebinding %q: %w", crb.Name, err)
	}

	err := wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		obj := crb.DeepCopy()
		if err := dm.client.Get(ctx, kutil.ObjectKeyFromObject(obj), obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	controllerutil.RemoveFinalizer(inst, lsv1alpha1.LandscaperDMFinalizer)
	if err := dm.Writer().UpdateInstallation(ctx, read_write_layer.W000013, inst); err != nil {
		return fmt.Errorf("unable to remove finalizer: %w", err)
	}
	return nil
}

// EnsureRBACRoles ensures that all needed rbac rules for the deployers are present on the system.
func (dm *DeployerManagement) EnsureRBACRoles(ctx context.Context) error {
	clusterrole := &rbacv1.ClusterRole{}
	clusterrole.Name = DeployerClusterRoleName

	if _, err := controllerutil.CreateOrUpdate(ctx, dm.client, clusterrole, func() error {
		// secrets to interact with the deploy items
		clusterrole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{lsv1alpha1.SchemeGroupVersion.Group},
				Resources: []string{"deployitems", "targets"},
				// update and patch is needed for interacting with annotations and finalizers
				Verbs: []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{lsv1alpha1.SchemeGroupVersion.Group},
				Resources: []string{"deployitems/status"},
				Verbs:     []string{"get", "watch", "list", "update", "patch"},
			},
			{
				APIGroups: []string{lsv1alpha1.SchemeGroupVersion.Group},
				Resources: []string{"contexts"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"events"},
				Verbs:     []string{"create", "get", "watch", "patch", "update"},
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to ensure cluster rabac role %q: %w", clusterrole.Name, err)
	}
	return nil
}

func (dm *DeployerManagement) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(dm.log, dm.client)
}

// FQName defines the fully qualified name for the resources created for a deployer installation.
func FQName(registration *lsv1alpha1.DeployerRegistration, env *lsv1alpha1.Environment) string {
	return registration.Name + "-" + env.Name
}

// FQNameFromName defines the fully qualified name for the resources created for a deployer installation.
func FQNameFromName(regName, envName string) string {
	return regName + "-" + envName
}
