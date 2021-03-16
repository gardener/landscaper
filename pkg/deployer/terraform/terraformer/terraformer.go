// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package terraformer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	kutils "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// Terraformer is the struct holding the configuration to only run
// a single independant instance of a Terraformer pod.
type Terraformer struct {
	log        logr.Logger
	kubeClient client.Client
	restConfig *rest.Config

	ConfigurationConfigMapName string
	StateConfigMapName         string
	TFVarsSecretName           string

	Labels map[string]string

	Name               string
	Namespace          string
	InitContainer      terraformv1alpha1.ContainerSpec
	TerraformContainer terraformv1alpha1.ContainerSpec
	LogLevel           string
}

// New creates a new Terraformer struct.
func New(log logr.Logger,
	kubeClient client.Client,
	restConfig *rest.Config,
	namespace, logLevel, itemNamespace, itemName string,
	initContainer, terraformContainer terraformv1alpha1.ContainerSpec) *Terraformer {
	id := generateId(itemName, itemNamespace)
	prefix := fmt.Sprintf("%s.", id)

	return &Terraformer{
		log:        log,
		kubeClient: kubeClient,
		restConfig: restConfig,

		Labels: map[string]string{
			LabelKeyItemName:      itemName,
			LabelKeyItemNamespace: itemNamespace,
		},

		ConfigurationConfigMapName: prefix + TerraformConfigSuffix,
		TFVarsSecretName:           prefix + TerraformTFVarsSuffix,
		StateConfigMapName:         prefix + TerraformStateSuffix,

		Name:               fmt.Sprintf("%s-%s", BaseName, id),
		Namespace:          namespace,
		InitContainer:      initContainer,
		TerraformContainer: terraformContainer,
		LogLevel:           logLevel,
	}
}

// generateId generates a single ID based on the name and the namespace
// of a DeployItem. This is then used to guarantee a unique identifier
// for the Terraformer RBAC resources and the pod name.
func generateId(itemName, itemNamespace string) string {
	key := fmt.Sprintf("%s/%s", itemNamespace, itemName)
	sum := sha256.Sum256([]byte(key))
	id := fmt.Sprintf("%x", sum)

	// Make sure we don't exceed 63 characters
	return id[:12]
}

// EnsurePod ensures a pod is created.
func (t *Terraformer) EnsurePod(ctx context.Context, command string, itemGeneration int64) (*corev1.Pod, error) {
	pod, err := t.createPod(ctx, command, itemGeneration)
	if err != nil {
		return pod, err
	}
	return t.waitForPodCreation(ctx, pod)
}

func (t *Terraformer) createPod(ctx context.Context, command string, itemGeneration int64) (*corev1.Pod, error) {

	sharedVolume := corev1.Volume{
		Name: "shared-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	sharedVolumeMount := corev1.VolumeMount{
		Name:      sharedVolume.Name,
		MountPath: SharedBasePath,
	}

	providersVolumeMount := corev1.VolumeMount{
		Name:      sharedVolume.Name,
		MountPath: TerraformerProvidersPath,
		SubPath:   SharedProvidersDirectory,
	}

	deployItemConfigurationVolume := corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: t.ConfigurationConfigMapName,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  DeployItemConfigurationFilename,
						Path: DeployItemConfigurationFilename,
					},
				},
				DefaultMode: nil,
				Optional:    nil,
			},
		},
	}
	deployItemConfigurationVolumeMount := corev1.VolumeMount{
		Name:      "config",
		ReadOnly:  true,
		MountPath: filepath.Dir(DeployItemConfigurationPath),
	}

	initContainer := corev1.Container{
		Name:            InitContainerName,
		Image:           t.InitContainer.Image,
		ImagePullPolicy: t.InitContainer.ImagePullPolicy,
		Env: []corev1.EnvVar{
			{
				Name:  DeployItemConfigurationPathName,
				Value: DeployItemConfigurationPath,
			},
			{
				Name:  RegistrySecretBasePathName,
				Value: RegistrySecretBasePath,
			},
			{
				Name:  TerraformSharedDirEnvVarName,
				Value: SharedBasePath,
			},
			{
				Name:  TerraformProvidersDirEnvVarName,
				Value: SharedProvidersPath,
			},
		},
		VolumeMounts: []corev1.VolumeMount{deployItemConfigurationVolumeMount, sharedVolumeMount},
	}

	mainContainer := corev1.Container{
		Name:            BaseName,
		Image:           t.TerraformContainer.Image,
		ImagePullPolicy: t.TerraformContainer.ImagePullPolicy,
		Command: []string{
			"/terraformer",
			command,
			"--zap-log-level=" + t.LogLevel,
			"--configuration-configmap-name=" + t.ConfigurationConfigMapName,
			"--state-configmap-name=" + t.StateConfigMapName,
			"--variables-secret-name=" + t.TFVarsSecretName,
		},
		VolumeMounts: []corev1.VolumeMount{providersVolumeMount, sharedVolumeMount},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1.5Gi"),
			},
		},
	}

	pod := &corev1.Pod{}
	pod.Name = t.Name
	pod.Namespace = t.Namespace
	pod.Labels = t.Labels
	pod.Labels[LabelKeyGeneration] = strconv.FormatInt(itemGeneration, 10)
	pod.Labels[LabelKeyCommand] = command

	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	pod.Spec.ServiceAccountName = t.Name
	pod.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(TerminationGracePeriodSeconds)

	pod.Spec.InitContainers = []corev1.Container{initContainer}
	pod.Spec.Containers = []corev1.Container{mainContainer}
	pod.Spec.Volumes = []corev1.Volume{deployItemConfigurationVolume, sharedVolume}

	if err := t.kubeClient.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// GetPod gets the pod. If the pod does not exist, it returns nil and the error.
func (t *Terraformer) GetPod(ctx context.Context) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	podKey := kutils.ObjectKey(t.Name, t.Namespace)
	podLog := t.log.WithValues("pod", podKey)

	podLog.Info("Get Terraformer pod")
	if err := t.kubeClient.Get(ctx, podKey, pod); err != nil {
		if apierrors.IsNotFound(err) {
			podLog.Info("No terraformer pod found")
			return nil, err
		}
		podLog.Error(err, "Error retrieving Terraformer pod")
		return nil, err
	}

	return pod, nil
}

// GetLogsAndDeletePod get the logs from a pod and delete it after.
// It returns the logs aggregated to other errors if the execution did not succeed.
func (t *Terraformer) GetLogsAndDeletePod(ctx context.Context, pod *corev1.Pod, command string, exitCode int32) error {
	var (
		// Contains all the errors for an execution.
		allErrs []error
	)

	podLog := t.log.WithValues("pod", kutils.ObjectKeyFromObject(pod))
	succeeded := exitCode == ExitCodeSucceeded
	if succeeded {
		podLog.Info("Terraformer pod completed successfully")
	} else {
		podLog.Info("Terraformer Pod finished with error", "exitCode", exitCode)
	}
	// Retrieve the logs of the plan/apply/destroy Pods
	t.log.V(1).Info("Fetching the logs of the Terraformer pod")
	logList, err := t.retrievePodLogs(ctx, pod)
	if err != nil {
		t.log.Error(err, "Could not retrieve the logs of the Terraformer pod")
		allErrs = append(allErrs, err)
		logList = map[string]string{}
	}

	if logs, ok := logList[pod.ObjectMeta.Name]; ok {
		podLog.V(1).Info(fmt.Sprintf("Logs of Terraformer Pod: %s", logs))
	}

	if err := t.kubeClient.Delete(ctx, pod); client.IgnoreNotFound(err) != nil {
		return err
	}

	// Evaluate whether the execution was successful or not
	t.log.Info("Terraformer execution has been completed")
	if !succeeded {
		errorMessage := fmt.Sprintf("Terraform execution for command '%s' could not be completed.", command)
		if terraformErrors := retrieveTerraformErrors(logList); terraformErrors != nil {
			errorMessage += fmt.Sprintf(" The following issues have been found in the logs:\n\n%s", strings.Join(terraformErrors, "\n\n"))
		}
		allErrs = append(allErrs, fmt.Errorf(errorMessage))
	}
	if len(allErrs) != 0 {
		return apimacherrors.NewAggregate(allErrs)
	}

	return nil
}

// retrievePodLogs fetches the logs of the Pod and returns them as a map whose
// keys are pod names and whose values are the corresponding logs.
func (t *Terraformer) retrievePodLogs(ctx context.Context, pod *corev1.Pod) (map[string]string, error) {
	logChan := make(chan map[string]string, 1)

	go func() {
		var logList = map[string]string{}
		name := pod.Name
		logs, err := func() ([]byte, error) {
			clientSet, err := kubernetes.NewForConfig(t.restConfig)
			if err != nil {
				return nil, err
			}
			request := clientSet.CoreV1().Pods(pod.Namespace).GetLogs(name, &corev1.PodLogOptions{})

			stream, err := request.Stream(ctx)
			if err != nil {
				return nil, err
			}
			defer func() { utilruntime.HandleError(stream.Close()) }()

			return ioutil.ReadAll(stream)
		}()

		if err != nil {
			t.log.Error(err, "Could not retrieve the logs of Terraformer pod", "pod", kutils.ObjectKeyFromObject(pod))
		}

		logList[name] = string(logs)
		logChan <- logList
	}()

	select {
	case result := <-logChan:
		return result, nil
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("timeout when reading the logs of all pods created by Terraformer")
	}
}

// EnsureCleanedUp deletes the Terraform pod, the Terraform configuration,
// the RBAC resources and wait until everything has been cleaned up.
func (t *Terraformer) EnsureCleanedUp(ctx context.Context) error {
	t.log.Info("Ensuring all terraformer Pods for the item have been deleted")
	podList, err := t.listTerraformerPods(ctx)
	if err != nil {
		return err
	}
	if err := t.deletePods(ctx, podList); err != nil {
		return err
	}
	if err := t.cleanUpRBAC(ctx); err != nil {
		return err
	}
	if err := t.cleanUpConfig(ctx); err != nil {
		return err
	}
	return t.waitForCleanEnvironment(ctx)
}

// listTerraformerPods lists all pods in the Terraformer namespace which have the Terraformer labels.
func (t *Terraformer) listTerraformerPods(ctx context.Context) (*corev1.PodList, error) {
	var podList = &corev1.PodList{}

	if err := t.kubeClient.List(ctx, podList,
		client.InNamespace(t.Namespace),
		client.MatchingLabels(t.Labels)); err != nil {
		return nil, err
	}
	return podList, nil
}

// deletePods delete all the pods for a given pod list.
func (t *Terraformer) deletePods(ctx context.Context, podList *corev1.PodList) error {
	t.log.Info("Deleting Terraformer pods")
	for _, pod := range podList.Items {
		t.log.V(1).Info("Deleting Terraformer pod", "name", pod.Name)
		err := t.kubeClient.Delete(ctx, &pod)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

// waitForCleanEnvironment waits until no Terraform Pod exist for the current instance
// of the Terraformer.
func (t *Terraformer) waitForCleanEnvironment(ctx context.Context) error {
	t.log.Info("Waiting for clean environment...")
	pollCtx, cancel := context.WithTimeout(ctx, DeadlineCleaning)
	defer cancel()

	return wait.PollImmediateUntil(5*time.Second, func() (done bool, err error) {
		podList, err := t.listTerraformerPods(pollCtx)
		if err != nil {
			return false, err
		}
		if len(podList.Items) > 0 {
			t.log.Info("Waiting until all Terraformer Pods have been cleaned up")
			return false, fmt.Errorf("at least one terraformer pod still exists: %s", podList.Items[0].Name)
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.TFVarsSecretName, t.Namespace), &corev1.Secret{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.ConfigurationConfigMapName, t.Namespace), &corev1.ConfigMap{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.StateConfigMapName, t.Namespace), &corev1.ConfigMap{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.Name, t.Namespace), &corev1.ServiceAccount{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.Name, t.Namespace), &rbacv1.Role{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		if err = t.kubeClient.Get(pollCtx, kutils.ObjectKey(t.Name, t.Namespace), &rbacv1.RoleBinding{}); client.IgnoreNotFound(err) != nil {
			return false, err
		}

		return true, nil
	}, pollCtx.Done())
}

// waitForPodCreation waits until the Terraformer pod is created.
func (t *Terraformer) waitForPodCreation(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	t.log.Info("Waiting for pod to be created...")
	pollCtx, cancel := context.WithTimeout(ctx, DeadlineCleaning)
	defer cancel()

	podKey := kutils.ObjectKey(t.Name, t.Namespace)
	err := wait.PollUntil(5*time.Second, func() (done bool, err error) {
		if err := t.kubeClient.Get(pollCtx, podKey, pod); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			t.log.Error(err, "unable to get pod", "pod", podKey.String())
			return false, err
		}
		return true, nil
	}, pollCtx.Done())
	if err != nil {
		return pod, err
	}
	return pod, nil

}
