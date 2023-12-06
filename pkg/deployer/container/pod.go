// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
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

// TargetSecretName generates the secret name for the imported secret.
// todo: use container identity
func TargetSecretName(deployItemNamespace, deployItemName string) string {
	return fmt.Sprintf("%s-%s-target", deployItemNamespace, deployItemName)
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

// DefaultLabels returns the default labels for a resource generated by the container deployer.
func DefaultLabels(deployerId, deployerName, diName, diNamespace string) map[string]string {
	return map[string]string{
		container.ContainerDeployerIDLabel:                  deployerId,
		container.ContainerDeployerNameLabel:                deployerName,
		container.ContainerDeployerDeployItemNameLabel:      diName,
		container.ContainerDeployerDeployItemNamespaceLabel: diNamespace,
	}
}

// InjectDefaultLabels injects default labels into the given object.
func InjectDefaultLabels(obj client.Object, defaultLabels map[string]string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range defaultLabels {
		labels[k] = v
	}
	obj.SetLabels(labels)
}

// PodOptions contains the configuration that is needed for the scheduled pod
type PodOptions struct {
	DeployerID string

	ProviderConfiguration             *containerv1alpha1.ProviderConfiguration
	InitContainer                     containerv1alpha1.ContainerSpec
	WaitContainer                     containerv1alpha1.ContainerSpec
	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName
	ConfigurationSecretName           string
	TargetSecretName                  string
	ImagePullSecret                   string
	BluePrintPullSecret               string
	ComponentDescriptorPullSecret     string

	UseOCM bool

	Name                 string
	Namespace            string
	DeployItemName       string
	DeployItemNamespace  string
	DeployItemGeneration int64

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

	targetVolume := corev1.Volume{
		Name: "target",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: opts.TargetSecretName,
			},
		},
	}
	targetInitVolumeMount := corev1.VolumeMount{
		Name:      targetVolume.Name,
		ReadOnly:  true,
		MountPath: container.TargetInitDir,
	}

	additionalInitEnvVars := []corev1.EnvVar{
		{
			Name:  container.ConfigurationPathName,
			Value: container.ConfigurationPath,
		},
		{
			Name:  container.DeployItemName,
			Value: opts.DeployItemName,
		},
		{
			Name:  container.DeployItemNamespaceName,
			Value: opts.DeployItemNamespace,
		},
		{
			Name:  container.RegistrySecretBasePathName,
			Value: container.RegistrySecretBasePath,
		},
		{
			Name:  container.UseOCMName,
			Value: fmt.Sprint(opts.UseOCM),
		},
	}
	additionalSidecarEnvVars := []corev1.EnvVar{
		{
			Name:  container.DeployItemName,
			Value: opts.DeployItemName,
		},
		{
			Name:  container.DeployItemNamespaceName,
			Value: opts.DeployItemNamespace,
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
		targetVolume,
	}

	initMounts := []corev1.VolumeMount{configurationVolumeMount, targetInitVolumeMount, initServiceAccountMount, sharedVolumeMount}

	for name, v := range map[string]string{
		"blueprint-pull-secret": opts.BluePrintPullSecret,
		"cd-pull-secret":        opts.ComponentDescriptorPullSecret} {
		if len(v) == 0 {
			continue
		}

		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: v,
				},
			},
		})

		initMounts = append(initMounts, corev1.VolumeMount{
			Name:      name,
			ReadOnly:  true,
			MountPath: filepath.Join(container.RegistrySecretBasePath, name),
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
	InjectDefaultLabels(pod, DefaultLabels(opts.DeployerID, opts.Name, opts.DeployItemName, opts.DeployItemNamespace))
	pod.Labels[container.ContainerDeployerDeployItemGenerationLabel] = strconv.Itoa(int(opts.DeployItemGeneration))
	pod.Finalizers = []string{container.ContainerDeployerFinalizer}

	pod.Spec.AutomountServiceAccountToken = pointer.Bool(false)
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.TerminationGracePeriodSeconds = pointer.Int64(300)
	pod.Spec.Volumes = volumes
	pod.Spec.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser:  pointer.Int64(1000),
		RunAsGroup: pointer.Int64(3000),
		FSGroup:    pointer.Int64(2000),
	}
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

// getPod returns the latest executed pod.
// Pods that have no finalizer are ignored.
func (c *Container) getPod(ctx context.Context) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := read_write_layer.ListPods(ctx, c.hostClient, podList, read_write_layer.R000077,
		client.InNamespace(c.Configuration.Namespace), client.MatchingLabels{
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
	var latest *corev1.Pod
	for _, pod := range podList.Items {
		// ignore pods with no finalizer as they are already reconciled and their state was persisted.
		if !controllerutil.ContainsFinalizer(&pod, container.ContainerDeployerFinalizer) {
			continue
		}
		if latest == nil {
			latest = pod.DeepCopy()
		}
		if pod.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = pod.DeepCopy()
		}
	}

	if latest == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    corev1.SchemeGroupVersion.Group,
			Resource: "Pod",
		}, c.DeployItem.Name)
	}

	return latest, nil
}

// EnsureServiceAccountsResult describes the result of the ensureServiceAccounts func
type EnsureServiceAccountsResult struct {
	InitContainerServiceAccountSecret types.NamespacedName
	WaitContainerServiceAccountSecret types.NamespacedName
}

// EnsureServiceAccounts ensures that the service accounts for the init and wait container are created
// and have the necessary permissions.
func EnsureServiceAccounts(ctx context.Context, hostClient client.Client, deployItem *lsv1alpha1.DeployItem, hostNamespace string, labels map[string]string) (*EnsureServiceAccountsResult, error) {
	var (
		res = &EnsureServiceAccountsResult{}
		log = logging.FromContextOrDiscard(ctx)
	)
	initSA := &corev1.ServiceAccount{}
	initSA.Name = InitContainerServiceAccountName(deployItem)
	initSA.Namespace = hostNamespace
	if _, err := controllerutil.CreateOrUpdate(ctx, hostClient, initSA, func() error {
		InjectDefaultLabels(initSA, labels)
		return nil
	}); err != nil {
		return nil, err
	}

	// create role and rolebindings for the init service account
	role := &rbacv1.Role{}
	role.Name = initSA.Name
	role.Namespace = initSA.Namespace
	_, err := controllerutil.CreateOrUpdate(ctx, hostClient, role, func() error {
		InjectDefaultLabels(role, labels)
		// need to read secrets to restore the state
		// deletion is needed for garbage collection
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "delete"},
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.RoleBinding{}
	rolebinding.Name = initSA.Name
	rolebinding.Namespace = initSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, hostClient, rolebinding, func() error {
		InjectDefaultLabels(rolebinding, labels)
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
		return nil, err
	}

	// wait for kubernetes to create the service accounts secrets
	res.InitContainerServiceAccountSecret, err = WaitAndGetServiceAccountSecret(ctx, log, hostClient, initSA, labels)
	if err != nil {
		return nil, err
	}
	waitSA := &corev1.ServiceAccount{}
	waitSA.Name = WaitContainerServiceAccountName(deployItem)
	waitSA.Namespace = hostNamespace
	if _, err := controllerutil.CreateOrUpdate(ctx, hostClient, waitSA, func() error {
		InjectDefaultLabels(waitSA, labels)
		return nil
	}); err != nil {
		return nil, err
	}

	// create role and rolebindings for the wait service account
	role = &rbacv1.Role{}
	role.Name = waitSA.Name
	role.Namespace = waitSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, hostClient, role, func() error {
		InjectDefaultLabels(role, labels)
		role.Rules = []rbacv1.PolicyRule{
			// we need a specific create secrets role as we cannot restrict the creation of secrets to a specific name
			// See https://kubernetes.io/docs/reference/access-authn-authz/rbac/
			// "You cannot restrict create or deletecollection requests by resourceName. For create, this limitation is because the object name is not known at authorization time."
			// the ait container needs permissions to write secrets for its state.
			{
				APIGroups: []string{corev1.SchemeGroupVersion.Group},
				Resources: []string{"secrets"},
				Verbs:     []string{"create", "update", "get", "list"},
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
		return nil, err
	}

	rolebinding = &rbacv1.RoleBinding{}
	rolebinding.Name = waitSA.Name
	rolebinding.Namespace = waitSA.Namespace
	_, err = controllerutil.CreateOrUpdate(ctx, hostClient, rolebinding, func() error {
		InjectDefaultLabels(rolebinding, labels)
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
		return nil, err
	}

	// wait for kubernetes to create the service accounts secrets
	res.WaitContainerServiceAccountSecret, err = WaitAndGetServiceAccountSecret(ctx, log, hostClient, waitSA, labels)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// WaitAndGetServiceAccountSecret waits until a service accounts secret is available and returns the secrets name.
func WaitAndGetServiceAccountSecret(ctx context.Context, log logging.Logger, c client.Client, serviceAccount *corev1.ServiceAccount, labels map[string]string) (types.NamespacedName, error) {
	secretKey := types.NamespacedName{}
	config := &rest.Config{}
	if err := kutil.AddServiceAccountToken(ctx, c, serviceAccount, config); err != nil {
		return secretKey, err
	}
	saKey := client.ObjectKeyFromObject(serviceAccount)
	secrets, err := kutil.GetSecretsForServiceAccount(ctx, c, saKey)
	if err != nil {
		log.Error(err, "unable to get secrets for service account", "serviceaccount", saKey.String())
		return secretKey, err
	}

	if len(secrets) == 0 {
		return secretKey, fmt.Errorf("no secret found for service account %s", saKey.String())
	}

	for _, secret := range secrets {
		InjectDefaultLabels(secret, labels)
		if err := c.Update(ctx, secret); err != nil {
			return secretKey, err
		}
	}
	secretKey = client.ObjectKeyFromObject(secrets[0])
	return secretKey, nil
}
