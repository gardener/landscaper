// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Agent is the internal landscaper agent that contains all landscaper specific code.
type Agent struct {
	log            logr.Logger
	config         config.AgentConfiguration
	lsClient       client.Client
	lsRestConfig   *rest.Config
	lsScheme       *runtime.Scheme
	hostClient     client.Client
	hostScheme     *runtime.Scheme
	hostRestConfig *rest.Config
}

// New creates a new agent.
func New(log logr.Logger,
	lsClient client.Client,
	lsRestConfig *rest.Config,
	lsScheme *runtime.Scheme,
	hostClient client.Client,
	hostRestConfig *rest.Config,
	hostScheme *runtime.Scheme,
	config config.AgentConfiguration) *Agent {
	return &Agent{
		log:            log,
		config:         config,
		lsClient:       lsClient,
		lsRestConfig:   lsRestConfig,
		lsScheme:       lsScheme,
		hostClient:     hostClient,
		hostRestConfig: hostRestConfig,
		hostScheme:     hostScheme,
	}
}

// EnsureLandscaperResources ensures that all landscaper resources
// like the Environment and the Target are registered in the landscaper cluster.
func (a *Agent) EnsureLandscaperResources(ctx context.Context, lsClient, hostClient client.Client) (*lsv1alpha1.Environment, error) {
	target, err := utils.NewTargetBuilder(string(lsv1alpha1.KubernetesClusterTargetType)).
		Config(lsv1alpha1.KubernetesClusterTargetConfig{
			Kubeconfig: lsv1alpha1.ValueRef{
				SecretRef: &lsv1alpha1.SecretReference{
					ObjectReference: lsv1alpha1.ObjectReference{
						Name:      a.TargetSecretName(),
						Namespace: a.config.Namespace,
					},
					Key: lsv1alpha1.DefaultKubeconfigKey,
				},
			},
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
		env.Spec.TargetSelectors = a.config.TargetSelectors
		if env.Spec.TargetSelectors == nil {
			env.Spec.TargetSelectors = DefaultTargetSelector(env.Name)
		}

		env.Spec.HostTarget.Annotations = map[string]string{
			lsv1alpha1.DeployerEnvironmentTargetAnnotationName: env.Name,
			lsv1alpha1.DeployerOnlyTargetAnnotationName:        "true",
		}
		env.Spec.HostTarget.TargetSpec = target.Spec
	}

	if err := lsClient.Get(ctx, kutil.ObjectKeyFromObject(env), env); err != nil {
		if apierrors.IsNotFound(err) {
			mutateFunc()
			if err := lsClient.Create(ctx, env); err != nil {
				return nil, err
			}
			return env, nil
		}
		return nil, err
	}

	if !env.DeletionTimestamp.IsZero() {
		// the environment has the deployer management finalizer if there are still deployer installation
		// therefore do nothing.
		if !controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer) {
			// cleanup resources but do not remove the finalizer
			// as we would otherwise just right directly reconcile a new environment.
			if err := a.RemoveHostResources(ctx, hostClient); err != nil {
				return nil, fmt.Errorf("unable to remove host resources: %w", err)
			}
			return env, nil
		}
		// still update the environment to reconcile possible new configurations.
	}

	mutateFunc()
	if err := lsClient.Update(ctx, env); err != nil {
		return nil, err
	}
	return env, nil
}

// EnsureHostResources ensures that all host resources
// like the Target secret are registered in the landscaper cluster.
// The function ensure the following resources:
// - the secret containing the kubeconfig for the host kubeconfig
func (a *Agent) EnsureHostResources(ctx context.Context, kubeClient client.Client) (*rest.Config, error) {
	// create a dedicated service account and rbac rules for the kubeconfig
	// Currently that kubeconfig has access to all resources as the deployers could install anything.
	// We might need to restrict that in the future but at least the access the be audited.
	sa := &corev1.ServiceAccount{}
	sa.Name = fmt.Sprintf("deployer-%s", a.config.Name)
	sa.Namespace = a.config.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, sa, func() error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create service account %q for deployer on host cluster: %w", sa.Name, err)
	}
	cr := DeployerClusterRole(a.config.Name)
	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, cr, func() error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create cluster role %q for deployer on host cluster: %w", cr.Name, err)
	}
	crb := DeployerClusterRoleBinding(sa, a.config.Name)
	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, crb, func() error {
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to create cluster role biinding %q for deployer on host cluster: %w", crb.Name, err)
	}

	hostRestConfig := rest.CopyConfig(a.hostRestConfig)
	if err := kutil.AddServiceAccountAuth(ctx, kubeClient, sa, hostRestConfig); err != nil {
		return nil, err
	}

	kubeconfigBytes, err := kutil.GenerateKubeconfigBytes(hostRestConfig)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	secret.Name = a.TargetSecretName()
	secret.Namespace = a.config.Namespace

	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DefaultKubeconfigKey: kubeconfigBytes,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return hostRestConfig, nil
}

// RemoveHostResources removes all resources created by the agent from the host.
func (a *Agent) RemoveHostResources(ctx context.Context, kubeClient client.Client) error {
	sa := &corev1.ServiceAccount{}
	sa.Name = fmt.Sprintf("deployer-%s", a.config.Name)
	sa.Namespace = a.config.Namespace
	cr := DeployerClusterRole(a.config.Name)
	crb := DeployerClusterRoleBinding(sa, a.config.Name)

	resources := []client.Object{sa, cr, crb}
	for _, obj := range resources {
		if err := kubeClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
	}
	err := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		for _, obj := range resources {
			if err := kubeClient.Get(ctx, kutil.ObjectKeyFromObject(obj), obj); err != nil {
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
	if req.Name != a.config.Name {
		return reconcile.Result{}, nil
	}
	env, err := a.EnsureLandscaperResources(ctx, a.lsClient, a.hostClient)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !env.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer) {
		return reconcile.Result{}, nil
	}
	if _, err := a.EnsureHostResources(ctx, a.hostClient); err != nil {
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
				{
					Key:      lsv1alpha1.DeployerOnlyTargetAnnotationName,
					Operator: selection.DoesNotExist,
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
		data, err := ioutil.ReadFile(restConfig.TLSClientConfig.CAFile)
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
	crb.Subjects = []rbacv1.Subject{
		{
			APIGroup:  "",
			Kind:      "ServiceAccount",
			Name:      sa.Name,
			Namespace: sa.Namespace,
		},
	}
	return crb
}
