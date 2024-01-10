// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

// Agent is the internal landscaper agent that contains all landscaper specific code.
type Agent struct {
	lsUncachedClient   client.Client
	lsCachedClient     client.Client
	hostUncachedClient client.Client
	hostCachedClient   client.Client
	config             config.AgentConfiguration
	lsRestConfig       *rest.Config
	lsScheme           *runtime.Scheme
	hostScheme         *runtime.Scheme
	hostRestConfig     *rest.Config
}

// New creates a new agent.
func New(lsUncachedClient, lsCachedClient, hostUncachedClient, hostCachedClient client.Client,
	lsRestConfig *rest.Config,
	lsScheme *runtime.Scheme,
	hostRestConfig *rest.Config,
	hostScheme *runtime.Scheme,
	config config.AgentConfiguration) *Agent {
	return &Agent{
		lsUncachedClient:   lsUncachedClient,
		lsCachedClient:     lsCachedClient,
		hostUncachedClient: hostUncachedClient,
		hostCachedClient:   hostCachedClient,
		config:             config,
		lsRestConfig:       lsRestConfig,
		lsScheme:           lsScheme,
		hostRestConfig:     hostRestConfig,
		hostScheme:         hostScheme,
	}
}

// EnsureLandscaperResources ensures that all landscaper resources
// like the Environment and the Target are registered in the landscaper cluster.
func (a *Agent) EnsureLandscaperResources(ctx context.Context) (*lsv1alpha1.Environment, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "EnsureLandscaperResources")
	target, err := utils.NewTargetBuilder(string(targettypes.KubernetesClusterTargetType)).
		SecretRef(&lsv1alpha1.LocalSecretReference{
			Name: a.TargetSecretName(),
			Key:  targettypes.DefaultKubeconfigKey,
		}).Build()
	if err != nil {
		return nil, err
	}

	clusterRestConfig, err := GenerateClusterRestConfig(a.lsRestConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to generate cluster rest config")
	}

	env := &lsv1alpha1.Environment{}
	env.Name = a.config.Name

	mutateFunc := func() {
		controllerutil.AddFinalizer(env, lsv1alpha1.LandscaperAgentFinalizer)
		env.Spec.Namespace = a.config.Namespace
		env.Spec.LandscaperClusterRestConfig = clusterRestConfig

		if len(a.config.TargetSelectors) == 0 {
			env.Spec.TargetSelectors = DefaultTargetSelector(env.Name)
		} else {
			env.Spec.TargetSelectors = a.config.TargetSelectors
		}

		env.Spec.HostTarget.Annotations = map[string]string{
			lsv1alpha1.DeployerEnvironmentTargetAnnotationName: env.Name,
			lsv1alpha1.DeployerOnlyTargetAnnotationName:        "true",
		}
		env.Spec.HostTarget.TargetSpec = target.Spec
	}

	if err := read_write_layer.GetEnv(ctx, a.lsUncachedClient, kutil.ObjectKeyFromObject(env), env, read_write_layer.R000038); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Environment not found, creating it")
			mutateFunc()
			if err := a.lsUncachedClient.Create(ctx, env); err != nil {
				return nil, err
			}
			return env, nil
		}
		return nil, err
	}

	if !env.DeletionTimestamp.IsZero() {
		logger.Debug("Environment has deletion timestamp", lc.KeyDeletionTimestamp, env.DeletionTimestamp.Time)
		// the environment has the deployer management finalizer if there are still deployer installation
		// therefore do nothing.
		if !controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer) {
			logger.Debug("Cleaning up resources")
			// cleanup resources but do not remove the finalizer
			// as we would otherwise just right directly reconcile a new environment.
			if err := a.RemoveHostResources(ctx); err != nil {
				return nil, fmt.Errorf("unable to remove host resources: %w", err)
			}
			return env, nil
		} else {
			logger.Debug("Environment still has Landscaper deployer management finalizer")
		}
		// still update the environment to reconcile possible new configurations.
	}

	logger.Info("Updating Environment")
	mutateFunc()
	if err := a.lsUncachedClient.Update(ctx, env); err != nil {
		return nil, err
	}
	return env, nil
}

// EnsureHostResources ensures that all host resources
// like the Target secret are registered in the landscaper cluster.
// The function ensure the following resources:
// - the secret containing the kubeconfig for the host kubeconfig
func (a *Agent) EnsureHostResources(ctx context.Context) (*rest.Config, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "EnsureHostResources")
	// create a dedicated service account and rbac rules for the kubeconfig
	// Currently that kubeconfig has access to all resources as the deployers could install anything.
	// We might need to restrict that in the future but at least the access the be audited.
	sa := &corev1.ServiceAccount{}
	sa.Name = fmt.Sprintf("deployer-%s", a.config.Name)
	sa.Namespace = a.config.Namespace
	logger.Info("Creating/Updating resource", lc.KeyResource, kutil.ObjectKeyFromObject(sa).String(), lc.KeyResourceKind, "ServiceAccount")
	if _, err := controllerutil.CreateOrUpdate(ctx, a.hostUncachedClient, sa, func() error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create service account %q for deployer on host cluster: %w", sa.Name, err)
	}
	cr := DeployerClusterRole(a.config.Name)
	logger.Info("Creating/Updating resource", lc.KeyResource, kutil.ObjectKeyFromObject(cr).String(), lc.KeyResourceKind, "ClusterRole")
	if _, err := controllerutil.CreateOrUpdate(ctx, a.hostUncachedClient, cr, func() error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create cluster role %q for deployer on host cluster: %w", cr.Name, err)
	}
	crb := DeployerClusterRoleBinding(sa, a.config.Name)
	logger.Info("Creating/Updating resource", lc.KeyResource, kutil.ObjectKeyFromObject(crb).String(), lc.KeyResourceKind, "ClusterRoleBinding")
	if _, err := controllerutil.CreateOrUpdate(ctx, a.hostUncachedClient, crb, func() error {
		crb.Subjects = DeployerClusterRoleBindingSubjects(sa)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create cluster role biinding %q for deployer on host cluster: %w", crb.Name, err)
	}

	hostRestConfig := rest.CopyConfig(a.hostRestConfig)
	if err := kutil.AddServiceAccountToken(ctx, a.hostUncachedClient, sa, hostRestConfig); err != nil {
		logger.Error(err, "unable to add a service account token", "service-account", sa.Name)
		return nil, err
	}

	kubeconfigBytes, err := kutil.GenerateKubeconfigBytes(hostRestConfig)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	secret.Name = a.TargetSecretName()
	secret.Namespace = a.config.LandscaperNamespace

	if _, err := controllerutil.CreateOrUpdate(ctx, a.lsUncachedClient, secret, func() error {
		secret.Data = map[string][]byte{
			targettypes.DefaultKubeconfigKey: kubeconfigBytes,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return hostRestConfig, nil
}

// RemoveHostResources removes all resources created by the agent from the host.
func (a *Agent) RemoveHostResources(ctx context.Context) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "RemoveHostResources")
	sa := &corev1.ServiceAccount{}
	sa.Name = fmt.Sprintf("deployer-%s", a.config.Name)
	sa.Namespace = a.config.Namespace
	sa.SetGroupVersionKind(schema.GroupVersionKind{})
	cr := DeployerClusterRole(a.config.Name)
	crb := DeployerClusterRoleBinding(sa, a.config.Name)

	resources := []client.Object{sa, cr, crb}
	for _, obj := range resources {
		if err := a.hostUncachedClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if logger.Enabled(logging.DEBUG) {
			log := logger.WithValues(lc.KeyResource, kutil.ObjectKeyFromObject(obj).String())
			gvk, err := apiutil.GVKForObject(obj, api.LandscaperScheme)
			if err != nil {
				log.Error(err, "unable to find GVK for object")
			} else {
				log.Debug("Waiting for resource to be deleted", lc.KeyResourceKind, gvk.Kind)
			}
		}
	}

	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 5*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		for _, obj := range resources {

			if err := read_write_layer.GetObject(ctx, a.hostUncachedClient, kutil.ObjectKeyFromObject(obj), obj, read_write_layer.R000039); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
			}
			return false, nil
		}
		return true, nil
	})
	return err
}

// Reconcile reconciles the environment and target on the landscaper.
func (a *Agent) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := logging.MustStartReconcileFromContext(ctx, req, nil)
	if req.Name != a.config.Name {
		logger.Info("Not responsible for this environment")
		return reconcile.Result{}, nil
	}
	logger.Info("Ensuring Landscaper resources")
	env, err := a.EnsureLandscaperResources(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !env.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer) {
		logger.Info("Environment is in deletion", lc.KeyDeletionTimestamp, env.DeletionTimestamp.Time)
		return reconcile.Result{}, nil
	}
	logger.Info("Ensuring host resources")
	if _, err := a.EnsureHostResources(ctx); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (a Agent) TargetSecretName() string {
	return a.config.Name + "-target-access"
}

// DefaultTargetSelector defines the default target selector.
func DefaultTargetSelector(envName string) []lsv1alpha1.TargetSelector {
	return []lsv1alpha1.TargetSelector{
		{
			Annotations: []lsv1alpha1.Requirement{
				{
					Key:      lsv1alpha1.DeployerEnvironmentTargetAnnotationName,
					Operator: selection.Equals,
					Values:   []string{envName},
				},
			},
		},
	}
}

// GenerateClusterRestConfig creates a cluster rest config from a rest config.
func GenerateClusterRestConfig(restConfig *rest.Config) (lsv1alpha1.ClusterRestConfig, error) {
	// the ca data has to be read from file for in-cluster configs
	caData := restConfig.CAData
	if len(restConfig.TLSClientConfig.CAFile) != 0 {
		data, err := os.ReadFile(restConfig.TLSClientConfig.CAFile)
		if err != nil {
			return lsv1alpha1.ClusterRestConfig{}, fmt.Errorf("unable to read ca data from %q: %w", restConfig.TLSClientConfig.CAFile, err)
		}
		caData = data
	}

	return lsv1alpha1.ClusterRestConfig{
		Host:    restConfig.Host,
		APIPath: restConfig.APIPath,
		TLSClientConfig: lsv1alpha1.TLSClientConfig{
			Insecure:   restConfig.Insecure,
			ServerName: restConfig.ServerName,
			CAData:     caData,
		},
	}, nil
}

// DeployerClusterRoleName is the prefix of the deployer cluster role in the host cluster.
// That role has access to the deployer needed artifacts like deploy items and secrets.
const DeployerClusterRoleName = "landscaper:agent:deployer"

// DeployerClusterRole returns the cluster role for the host cluster.
// The cluster role is assigned to the service account that is used for the host target.
func DeployerClusterRole(envName string) *rbacv1.ClusterRole {
	cr := &rbacv1.ClusterRole{}
	cr.Name = DeployerClusterRoleName
	cr.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
	}
	return cr
}

// DeployerClusterRoleBinding returns the cluster role for the host cluster.
// The cluster role is assigned to the service account that is used for the host target.
func DeployerClusterRoleBinding(sa *corev1.ServiceAccount, envName string) *rbacv1.ClusterRoleBinding {
	crb := &rbacv1.ClusterRoleBinding{}
	crb.Name = DeployerClusterRoleName + ":" + envName
	crb.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Kind:     "ClusterRole",
		Name:     DeployerClusterRoleName,
	}
	crb.Subjects = DeployerClusterRoleBindingSubjects(sa)
	return crb
}

func DeployerClusterRoleBindingSubjects(sa *corev1.ServiceAccount) []rbacv1.Subject {
	return []rbacv1.Subject{
		{
			APIGroup:  "",
			Kind:      "ServiceAccount",
			Name:      sa.Name,
			Namespace: sa.Namespace,
		},
	}
}
