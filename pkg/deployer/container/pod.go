// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// PodTokenPath is the path in the pod that contains the service account token.
const PodTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// InitContainerServiceAccountName generates the service account name for the init container
func InitContainerServiceAccountName(di *lsv1alpha1.DeployItem) string {
	return fmt.Sprintf("%s-init", di.Name)
}

// WaitContainerServiceAccountName generates the service account name for the wait container
func WaitContainerServiceAccountName(di *lsv1alpha1.DeployItem) string {
	return fmt.Sprintf("%s-wait", di.Name)
}

// ExportSecretName generates the secret name for the exported secret
func ExportSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-export", deployItemNamespace, deployItemName)
}

// DeployItemExportSecretName generates the secret name for the exported secret
func DeployItemExportSecretName(deployItemName string) string {
	return fmt.Sprintf("%s-export", deployItemName)
}

// ConfigurationSecretName generates the secret name for the imported secret.
// todo: use container identity
func ConfigurationSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-config", deployItemNamespace, deployItemName)
}

// ImagePullSecretName generates the secret name for the image pull secret.
// todo: use container identity
func ImagePullSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-imgpullsec", deployItemNamespace, deployItemName)
}

// BluePrintPullSecretName generates the secret name for the image pull secret.
// todo: use container identity
func BluePrintPullSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-bppullsec", deployItemNamespace, deployItemName)
}

// ComponentDescriptorPullSecretName generates the secret name for the image pull secret.
// todo: use container identity
func ComponentDescriptorPullSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-cdpullsec", deployItemNamespace, deployItemName)
}

// PodOptions contains the configuration that is needed for the scheduled pod
type PodOptions struct {
	ProviderConfiguration             *containerv1alpha1.ProviderConfiguration
	InitContainer                     containerv1alpha1.ContainerSpec
	WaitContainer                     containerv1alpha1.ContainerSpec
	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName
	ConfigurationSecretName           string
	ImagePullSecret                   string
	BluePrintPullSecret               string
	ComponentDescriptorPullSecret     string

	Name                string
	Namespace           string
	DeployItemName      string
	DeployItemNamespace string

	Operation       container.OperationType
	encBlueprintRef []byte

	Debug bool
}

// Complete completes the the Blueprint provider configuration
func (o *PodOptions) Complete() error {
	if o.ProviderConfiguration.Blueprint != nil {
		raw, err := json.Marshal(o.ProviderConfiguration.Blueprint)
		if err != nil {
			return err
		}
		o.encBlueprintRef = raw
	}
	return nil
}

func generatePod(opts PodOptions) (*corev1.Pod, error) {
	if err := opts.Complete(); err != nil {
		return nil, err
	}

	sharedVolume := corev1.Volume{
		Name: "shared-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	sharedVolumeMount := corev1.VolumeMount{
		Name:      sharedVolume.Name,
		MountPath: container.SharedBasePath,
	}

	initServiceAccountVolume := corev1.Volume{
		Name: "serviceaccount-init",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: opts.InitContainerServiceAccountSecret.Name,
			},
		},
	}
	initServiceAccountMount := corev1.VolumeMount{
		Name:      initServiceAccountVolume.Name,
		ReadOnly:  true,
		MountPath: filepath.Dir(PodTokenPath),
	}

	waitServiceAccountMount := corev1.VolumeMount{
		Name:      "serviceaccount-wait",
		ReadOnly:  true,
		MountPath: filepath.Dir(PodTokenPath),
	}
	waitServiceAccountVolume := corev1.Volume{
		Name: "serviceaccount-wait",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: opts.WaitContainerServiceAccountSecret.Name,
			},
		},
	}

	configurationVolume := corev1.Volume{
		Name: "configuration",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: opts.ConfigurationSecretName,
			},
		},
	}
	configurationVolumeMount := corev1.VolumeMount{
		Name:      configurationVolume.Name,
		ReadOnly:  true,
		MountPath: filepath.Dir(container.ConfigurationPath),
	}

	additionalInitEnvVars := []corev1.EnvVar{
		{
			Name:  container.ConfigurationPathName,
			Value: container.ConfigurationPath,
		},
		{
			Name:  container.DeployItemName,
			Value: opts.Name,
		},
		{
			Name:  container.DeployItemNamespaceName,
			Value: opts.Namespace,
		},
		{
			Name:  container.RegistrySecretBasePathName,
			Value: container.RegistrySecretBasePath,
		},
	}
	additionalSidecarEnvVars := []corev1.EnvVar{
		{
			Name:  container.DeployItemName,
			Value: opts.Name,
		},
		{
			Name:  container.DeployItemNamespaceName,
			Value: opts.Namespace,
		},
	}
	additionalEnvVars := []corev1.EnvVar{
		{
			Name:  container.OperationName,
			Value: string(opts.Operation),
		},
	}

	volumes := []corev1.Volume{
		initServiceAccountVolume,
		waitServiceAccountVolume,
		sharedVolume,
		configurationVolume,
	}

	initMounts := []corev1.VolumeMount{configurationVolumeMount, initServiceAccountMount, sharedVolumeMount}

	for _, v := range []string{opts.BluePrintPullSecret, opts.ComponentDescriptorPullSecret} {
		if len(v) == 0 {
			continue
		}

		volumes = append(volumes, corev1.Volume{
			Name: v,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: v,
				},
			},
		})

		initMounts = append(initMounts, corev1.VolumeMount{
			Name:      v,
			ReadOnly:  true,
			MountPath: filepath.Join(container.RegistrySecretBasePath, v),
		})
	}

	initContainer := corev1.Container{
		Name:                     container.InitContainerName,
		Image:                    opts.InitContainer.Image,
		Command:                  opts.InitContainer.Command,
		Args:                     opts.InitContainer.Args,
		Env:                      append(container.DefaultEnvVars, additionalInitEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          opts.InitContainer.ImagePullPolicy,
		VolumeMounts:             initMounts,
	}

	waitContainer := corev1.Container{
		Name:                     container.WaitContainerName,
		Image:                    opts.WaitContainer.Image,
		Command:                  opts.WaitContainer.Command,
		Args:                     opts.WaitContainer.Args,
		Env:                      append(container.DefaultEnvVars, additionalSidecarEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          opts.WaitContainer.ImagePullPolicy,
		VolumeMounts: []corev1.VolumeMount{
			waitServiceAccountMount,
			sharedVolumeMount,
		},
	}

	mainContainer := corev1.Container{
		Name:                     container.MainContainerName,
		Image:                    opts.ProviderConfiguration.Image,
		Command:                  opts.ProviderConfiguration.Command,
		Args:                     opts.ProviderConfiguration.Args,
		Env:                      append(container.DefaultEnvVars, additionalEnvVars...),
		Resources:                corev1.ResourceRequirements{},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		VolumeMounts:             []corev1.VolumeMount{sharedVolumeMount},
	}

	if opts.Debug {
		initContainer.ImagePullPolicy = corev1.PullAlways
		waitContainer.ImagePullPolicy = corev1.PullAlways
	}

	pod := &corev1.Pod{}
	pod.GenerateName = opts.Name + "-"
	pod.Namespace = opts.Namespace
	pod.Labels = map[string]string{
		container.ContainerDeployerNameLabel:                opts.Name,
		container.ContainerDeployerDeployItemNameLabel:      opts.DeployItemName,
		container.ContainerDeployerDeployItemNamespaceLabel: opts.DeployItemNamespace,
	}
	pod.Finalizers = []string{container.ContainerDeployerFinalizer}

	pod.Spec.AutomountServiceAccountToken = pointer.BoolPtr(false)
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(300)
	pod.Spec.Volumes = volumes
	pod.Spec.InitContainers = []corev1.Container{initContainer}
	pod.Spec.Containers = []corev1.Container{mainContainer, waitContainer}
	if len(opts.ImagePullSecret) != 0 {
		pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: opts.ImagePullSecret,
			},
		}
	}
	return pod, nil
}

func (c *Container) getPod(ctx context.Context) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := c.hostClient.List(ctx, podList, client.InNamespace(c.Configuration.Namespace), client.MatchingLabels{
		container.ContainerDeployerDeployItemNameLabel:      c.DeployItem.Name,
		container.ContainerDeployerDeployItemNamespaceLabel: c.DeployItem.Namespace,
	}); err != nil {
		return nil, err
	}

	if len(podList.Items) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    corev1.SchemeGroupVersion.Group,
			Resource: "Pod",
		}, c.DeployItem.Name)
	}

	// only return latest pod and ignore previous runs
	latest := podList.Items[0]
	for _, pod := range podList.Items {
		if pod.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = pod
		}
	}

	return &latest, nil
}

// ensureServiceAccounts ensures that the service accounts for the init and wait container are created
// and have the necessary permissions.
func (c *Container) ensureServiceAccounts(ctx context.Context) error {
	initSA := &corev1.ServiceAccount{}
	initSA.Name = InitContainerServiceAccountName(c.DeployItem)
	initSA.Namespace = c.Configuration.Namespace
	if _, err := kutil.CreateOrUpdate(ctx, c.hostClient, initSA, func() error { return nil }); err != nil {
		return err
	}

	// create role and rolebindings for the init service account
	role := &rbacv1.Role{}
	role.Name = initSA.Name
	role.Namespace = initSA.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, c.hostClient, role, func() error {
		// todo: consider different namespace of secrets
		if len(c.ProviderConfiguration.RegistryPullSecrets) != 0 {
			rule := rbacv1.PolicyRule{
				APIGroups:     []string{corev1.SchemeGroupVersion.Group},
				Resources:     []string{"secrets"},
				Verbs:         []string{"get"},
				ResourceNames: []string{},
			}
			for _, refs := range c.ProviderConfiguration.RegistryPullSecrets {
				rule.ResourceNames = append(rule.ResourceNames, refs.Name)
			}
			role.Rules = append(role.Rules, rule)
		}
		return nil
	})
	if err != nil {
		return err
	}

	rolebinding := &rbacv1.RoleBinding{}
	rolebinding.Name = initSA.Name
	rolebinding.Namespace = initSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, c.hostClient, rolebinding, func() error {
		rolebinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     role.Name,
		}
		rolebinding.Subjects = []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      initSA.Name,
				Namespace: initSA.Namespace,
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	// wait for kubernetes to create the service accounts secrets
	c.InitContainerServiceAccountSecret, err = WaitAndGetServiceAccountSecret(ctx, c.log, c.hostClient, initSA)
	if err != nil {
		return err
	}
	waitSA := &corev1.ServiceAccount{}
	waitSA.Name = WaitContainerServiceAccountName(c.DeployItem)
	waitSA.Namespace = c.Configuration.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostClient, waitSA, func() error { return nil }); err != nil {
		return err
	}

	// create role and rolebindings for the wait service account
	role = &rbacv1.Role{}
	role.Name = waitSA.Name
	role.Namespace = waitSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, c.hostClient, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{lsv1alpha1.SchemeGroupVersion.Group},
				Resources:     []string{"deployitems", "deployitems/status"},
				Verbs:         []string{"get", "update"},
				ResourceNames: []string{c.DeployItem.Name},
			},
			// we need a specific create secrets role as we cannot restrict the creation of a secret to a specific name
			// See https://kubernetes.io/docs/reference/access-authn-authz/rbac/
			// "You cannot restrict create or deletecollection requests by resourceName. For create, this limitation is because the object name is not known at authorization time."
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{corev1.SchemeGroupVersion.Group},
				Resources:     []string{"secrets"},
				Verbs:         []string{"update", "get"},
				ResourceNames: []string{ExportSecretName(c.DeployItem.Namespace, c.DeployItem.Name)},
			},
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"pods"},
				Verbs:     []string{"get"},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	rolebinding = &rbacv1.RoleBinding{}
	rolebinding.Name = waitSA.Name
	rolebinding.Namespace = waitSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, c.hostClient, rolebinding, func() error {
		rolebinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     role.Name,
		}
		rolebinding.Subjects = []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      waitSA.Name,
				Namespace: waitSA.Namespace,
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	// wait for kubernetes to create the service accounts secrets
	c.WaitContainerServiceAccountSecret, err = WaitAndGetServiceAccountSecret(ctx, c.log, c.hostClient, waitSA)
	if err != nil {
		return err
	}
	return nil
}

// WaitAndGetServiceAccountSecret waits until a service accounts secret is available and returns the secrets name.
func WaitAndGetServiceAccountSecret(ctx context.Context, log logr.Logger, c client.Client, serviceAccount *corev1.ServiceAccount) (types.NamespacedName, error) {
	timeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var secretKey types.NamespacedName
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		sa := &corev1.ServiceAccount{}
		saKey := kutil.ObjectKey(serviceAccount.Name, serviceAccount.Namespace)
		if err := c.Get(timeout, saKey, sa); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			log.Error(err, "unable to get service account", "serviceaccount", saKey.String())
			return false, nil
		}
		if len(sa.Secrets) == 0 {
			return false, nil
		}

		secret := &corev1.Secret{}
		secretKey = kutil.ObjectKey(sa.Secrets[0].Name, sa.Namespace)
		if err := c.Get(ctx, secretKey, secret); err != nil {
			if apierrors.IsNotFound(err) {
				return false, err
			}
			log.Error(err, "unable to get service account secret", "secret", secretKey.String())
			return false, nil
		}
		if secret.Type != corev1.SecretTypeServiceAccountToken {
			return false, fmt.Errorf("expected secret of type %s but found %s", corev1.SecretTypeServiceAccountToken, secret.Type)
		}

		return true, nil
	}, timeout.Done())
	if err != nil {
		return secretKey, err
	}
	return secretKey, nil
}
