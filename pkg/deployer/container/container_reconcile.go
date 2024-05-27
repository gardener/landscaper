// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	dockerreference "github.com/containerd/containerd/reference/docker"
	dockerconfig "github.com/docker/cli/cli/config"
	dockerconfigfile "github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/gardener/component-cli/ociclient/credentials"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	"github.com/gardener/landscaper/pkg/deployerlegacy"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Reconcile handles the reconcile flow for a container deploy item.
// todo: do retries on failure: difference between main container failure and init/wait container failure
func (c *Container) Reconcile(ctx context.Context, operation container.OperationType) error {
	if _, err := timeout.TimeoutExceeded(ctx, c.DeployItem, TimeoutCheckpointContainerStartReconcile); err != nil {
		return err
	}

	pod, err := c.getPod(ctx)
	logger := logging.FromContextOrDiscard(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		return lserrors.NewWrappedError(err,
			"Reconcile", "FetchRunningPod", err.Error())
	}

	lsWriter := read_write_layer.NewWriter(c.lsUncachedClient)

	// do nothing if the pod is still running
	if pod != nil {
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodUnknown {
			if err := c.collectAndSetPodStatus(pod, false); err != nil {
				return lserrors.NewWrappedError(err,
					"Reconcile", "UpdatePodStatus", err.Error())
			}
			// check if pod is in error state
			if err := podIsInErrorState(pod); err != nil {
				lsv1alpha1helper.SetDeployItemToFailed(c.DeployItem)
				if err := lsWriter.UpdateDeployItemStatus(ctx, read_write_layer.W000055, c.DeployItem); err != nil {
					return err // returns the error and retry
				}

				// only cleanup the pod if the error messages could be collected
				if err := c.CleanupPod(ctx, pod); err != nil {
					return err
				}
				return err
			}
			c.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing
			return nil
		}
	}

	if c.shouldRunNewPod(ctx, pod) {
		operationName := "DeployPod"

		// before we start syncing lets read the current deploy item from the server
		oldDeployItem := &lsv1alpha1.DeployItem{}
		if err := read_write_layer.GetDeployItem(ctx, c.lsUncachedClient, kutil.ObjectKey(c.DeployItem.GetName(),
			c.DeployItem.GetNamespace()), oldDeployItem, read_write_layer.R000027); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "FetchDeployItem", err.Error())
		}
		defaultLabels := DefaultLabels(c.Configuration.Identity, c.DeployItem.Name, c.DeployItem.Name, c.DeployItem.Namespace)

		if err := c.SyncConfiguration(ctx, defaultLabels); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "SyncConfiguration", err.Error())
		}

		if err := c.SyncTarget(ctx, defaultLabels); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "SyncTarget", err.Error())
		}

		if err := c.SyncOCMConfiguration(ctx, defaultLabels); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "SyncOCMConfiguration", err.Error())
		}

		imagePullSecret, blueprintSecret, componentDescriptorSecret, err := c.parseAndSyncSecrets(ctx, defaultLabels)
		if err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "ParseAndSyncSecrets", err.Error())
		}
		// ensure new pod
		serviceAccountSecrets, err := EnsureServiceAccounts(ctx, c.hostUncachedClient, c.DeployItem, c.Configuration.Namespace, defaultLabels)
		if err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "EnsurePodRBAC", err.Error())
		}
		c.InitContainerServiceAccountSecret, c.WaitContainerServiceAccountSecret = serviceAccountSecrets.InitContainerServiceAccountSecret, serviceAccountSecrets.WaitContainerServiceAccountSecret

		c.ProviderStatus = &containerv1alpha1.ProviderStatus{}
		podOpts := PodOptions{
			DeployerID: c.Configuration.Identity,

			ProviderConfiguration:             c.ProviderConfiguration,
			InitContainer:                     c.Configuration.InitContainer,
			WaitContainer:                     c.Configuration.WaitContainer,
			InitContainerServiceAccountSecret: c.InitContainerServiceAccountSecret,
			WaitContainerServiceAccountSecret: c.WaitContainerServiceAccountSecret,
			ConfigurationSecretName:           ConfigurationSecretName(c.DeployItem.Namespace, c.DeployItem.Name),
			TargetSecretName:                  TargetSecretName(c.DeployItem.Namespace, c.DeployItem.Name),

			ImagePullSecret:               imagePullSecret,
			BluePrintPullSecret:           blueprintSecret,
			ComponentDescriptorPullSecret: componentDescriptorSecret,

			OCMConfigConfigMapName: OCMConfigConfigMapName(c.DeployItem.Namespace, c.DeployItem.Name),
			UseOCM:                 c.Context.UseOCM,

			Name:                 c.DeployItem.Name,
			Namespace:            c.Configuration.Namespace,
			DeployItemName:       c.DeployItem.Name,
			DeployItemNamespace:  c.DeployItem.Namespace,
			DeployItemGeneration: c.DeployItem.Generation,

			Operation: operation,
			Debug:     true,
		}
		pod, err := generatePod(podOpts)
		if err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "PodGeneration", err.Error())
		}

		if err := c.hostUncachedClient.Create(ctx, pod); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "CreatePod", err.Error())
		}

		// update status
		c.ProviderStatus.LastOperation = string(operation)
		if err := c.collectAndSetPodStatus(pod, false); err != nil {
			return lserrors.NewWrappedError(err,
				operationName, "UpdatePodStatus", err.Error())
		}

		c.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing
		if operation == container.OperationDelete {
			c.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Deleting
		}

		if err := lsWriter.UpdateDeployItemStatus(ctx, read_write_layer.W000063, c.DeployItem); err != nil {
			return lserrors.NewWrappedError(err, operationName, "UpdateDeployItemStatus", err.Error())
		}

		if lsv1alpha1helper.HasOperation(c.DeployItem.ObjectMeta, lsv1alpha1.ReconcileOperation) {
			delete(c.DeployItem.Annotations, lsv1alpha1.OperationAnnotation)
			if err := lsWriter.UpdateDeployItem(ctx, read_write_layer.W000039, c.DeployItem); err != nil {
				return lserrors.NewWrappedError(err, operationName, "RemoveReconcileAnnotation", err.Error())
			}
		}
		return nil
	}

	operationName := "Complete"
	if pod != nil {
		podSucceeded := pod.Status.Phase == corev1.PodSucceeded
		if podSucceeded {
			if err := c.SyncExport(ctx); err != nil {
				return lserrors.NewWrappedError(err,
					operationName, "SyncExport", err.Error())
			}
		} else if pod.Status.Phase == corev1.PodFailed {
			lsv1alpha1helper.SetDeployItemToFailed(c.DeployItem)
		}

		c.ProviderStatus.LastOperation = string(operation)
		if err := c.collectAndSetPodStatus(pod, podSucceeded); err != nil {
			return lserrors.NewWrappedError(err,
				"Reconcile", "UpdatePodStatus", err.Error())
		}

		// write status to ensure podStatus is saved before deleting the pod
		if err := lsWriter.UpdateDeployItemStatus(ctx, read_write_layer.W000031, c.DeployItem); err != nil {
			return lserrors.NewWrappedError(err, operationName, "UpdateDeployItemStatus", err.Error())
		}

		// only remove the finalizer if we get the status of the pod
		logger.Debug("Deleting pod, as it has finished", "podStatus", pod.Status.Phase)
		if err := c.CleanupPod(ctx, pod); err != nil {
			return err
		}
	}
	if c.ProviderStatus != nil && c.ProviderStatus.PodStatus != nil && c.ProviderStatus.PodStatus.LastSuccessfulJobID != nil && *c.ProviderStatus.PodStatus.LastSuccessfulJobID == c.DeployItem.Status.JobID {
		logger.Debug("Setting phase to 'Succeeded', because pod was seen successfully finished for current jobID", lc.KeyJobID, c.DeployItem.Status.JobID)
		c.DeployItem.Status.Phase = lsv1alpha1.DeployItemPhases.Succeeded
	}
	return nil
}

// collectAndSetPodStatus the pod status and updates the container provider status
func (c *Container) collectAndSetPodStatus(pod *corev1.Pod, updateLastSuccessfulJobID bool) error {
	c.DeployItem.Status.Conditions = setConditionsFromPod(pod, c.DeployItem.Status.Conditions)
	var jobID *string
	if updateLastSuccessfulJobID {
		jobID = ptr.To[string](c.DeployItem.Status.JobID)
	}
	if err := setStatusFromPod(pod, c.ProviderStatus, jobID); err != nil {
		return err
	}

	encStatus, err := kutil.ConvertToRawExtension(c.ProviderStatus, Scheme)
	if err != nil {
		return err
	}
	c.DeployItem.Status.ProviderStatus = encStatus
	return nil
}

func (c *Container) shouldRunNewPod(ctx context.Context, pod *corev1.Pod) bool {
	// if there is already a pod we need to be sure that the current observed generation is not already run.
	genString := ""
	if pod != nil {
		ok := false
		if genString, ok = pod.Labels[container.ContainerDeployerDeployItemGenerationLabel]; ok {
			gen, err := strconv.Atoi(genString)
			if err == nil {
				if int64(gen) == c.DeployItem.Generation {
					return false
				}
			}
		}
	}
	logger, _ := logging.FromContextOrNew(ctx, nil)
	if c.ProviderStatus != nil && c.ProviderStatus.PodStatus != nil && c.ProviderStatus.PodStatus.LastSuccessfulJobID != nil && *c.ProviderStatus.PodStatus.LastSuccessfulJobID == c.DeployItem.Status.JobID {
		logger.Debug("No new pod required, pod for current JobID has successfully finished", lc.KeyJobID, c.DeployItem.Status.JobID, lc.KeyJobIDFinished, c.DeployItem.Status.JobIDFinished)
		return false
	}
	if c.DeployItem.Status.Phase == lsv1alpha1.DeployItemPhases.Init {
		var lsji *string
		if c.ProviderStatus != nil && c.ProviderStatus.PodStatus != nil {
			lsji = c.ProviderStatus.PodStatus.LastSuccessfulJobID
		}
		logger.Debug("newRootLogger pod required", "podExists", pod != nil, "podGenerationLabel", genString, lc.KeyDeployItemPhase, c.DeployItem.Status.Phase, "podStatusLastSuccessfulJobID", lsji)
		return true
	}
	return false
}

func setStatusFromPod(pod *corev1.Pod, providerStatus *containerv1alpha1.ProviderStatus, currentJobID *string) error {
	podStatus := &containerv1alpha1.PodStatus{
		PodName: pod.Name,
		LastRun: &pod.CreationTimestamp,
	}
	if providerStatus.PodStatus != nil {
		podStatus = providerStatus.PodStatus
	}

	if currentJobID != nil {
		podStatus.LastSuccessfulJobID = currentJobID
	}

	if mainContainerStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName); err == nil {
		podStatus.ContainerStatus = convertCoreContainerStatusToV1alpha1Container(mainContainerStatus)
	}
	if initContainerStatus, err := kutil.GetStatusForContainer(pod.Status.InitContainerStatuses, container.InitContainerName); err == nil {
		podStatus.InitContainerStatus = convertCoreContainerStatusToV1alpha1Container(initContainerStatus)
	}
	if waitContainerStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.WaitContainerName); err == nil {
		podStatus.WaitContainerStatus = convertCoreContainerStatusToV1alpha1Container(waitContainerStatus)
	}

	providerStatus.PodStatus = podStatus
	return nil
}

// convertCoreContainerStatusToV1alpha1Container converts a kubernetes container status into a container deployer container status.
func convertCoreContainerStatusToV1alpha1Container(containerStatus corev1.ContainerStatus) containerv1alpha1.ContainerStatus {
	cs := containerv1alpha1.ContainerStatus{}
	cs.Image = containerStatus.Image
	cs.ImageID = containerStatus.ImageID

	if containerStatus.State.Waiting != nil {
		cs.Message = containerStatus.State.Waiting.Message
		cs.Reason = containerStatus.State.Waiting.Reason
	}
	if containerStatus.State.Running != nil {
		cs.Reason = "Running"
	}
	if containerStatus.State.Terminated != nil {
		cs.Reason = containerStatus.State.Terminated.Reason
		cs.Message = containerStatus.State.Terminated.Message
		cs.ExitCode = &containerStatus.State.Terminated.ExitCode
	}

	return cs
}

func setConditionsFromPod(pod *corev1.Pod, conditions []lsv1alpha1.Condition) []lsv1alpha1.Condition {
	initStatus, err := kutil.GetStatusForContainer(pod.Status.InitContainerStatuses, container.InitContainerName)
	if err == nil {
		cond := lsv1alpha1helper.GetOrInitCondition(conditions, container.InitContainerConditionType)
		if initStatus.State.Waiting != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing, initStatus.State.Waiting.Reason, initStatus.State.Waiting.Message)
		}
		if initStatus.State.Running != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing,
				"Pod running",
				fmt.Sprintf("Pod started running at %s", initStatus.State.Running.StartedAt.String()))
		}
		if initStatus.State.Terminated != nil {
			if initStatus.State.Terminated.ExitCode == 0 {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionTrue,
					"ContainerSucceeded",
					"Container successfully finished")
			} else {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionFalse,
					initStatus.State.Terminated.Reason,
					initStatus.State.Terminated.Message)
			}
		}
		conditions = lsv1alpha1helper.MergeConditions(conditions, cond)
	}

	waitStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.WaitContainerName)
	if err == nil {
		cond := lsv1alpha1helper.GetOrInitCondition(conditions, container.WaitContainerConditionType)
		if waitStatus.State.Waiting != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing, waitStatus.State.Waiting.Reason, waitStatus.State.Waiting.Message)
		}
		if waitStatus.State.Running != nil {
			cond = lsv1alpha1helper.UpdatedCondition(cond,
				lsv1alpha1.ConditionProgressing,
				"Pod running",
				fmt.Sprintf("Pod started running at %s", waitStatus.State.Running.StartedAt.String()))
		}
		if waitStatus.State.Terminated != nil {
			if waitStatus.State.Terminated.ExitCode == 0 {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionTrue,
					"ContainerSucceeded",
					"Container successfully finished")
			} else {
				cond = lsv1alpha1helper.UpdatedCondition(cond,
					lsv1alpha1.ConditionFalse,
					waitStatus.State.Terminated.Reason,
					waitStatus.State.Terminated.Message)
			}
		}
		conditions = lsv1alpha1helper.MergeConditions(conditions, cond)
	}

	return conditions
}

// podIsInErrorState detects specific issues with a pod when the pod is in pending or progressing state.
// This should ensure that errors are detected earlier.
// currently we only detect ErrImagePull
func podIsInErrorState(pod *corev1.Pod) error {
	initStatus, err := kutil.GetStatusForContainer(pod.Status.InitContainerStatuses, container.InitContainerName)
	if err == nil {
		if initStatus.State.Waiting != nil {

			if utils.StringIsOneOf(initStatus.State.Waiting.Reason,
				kutil.ErrImagePull,
				kutil.ErrInvalidImageName,
				kutil.ErrRegistryUnavailable,
				kutil.ErrImageNeverPull,
				kutil.ErrImagePullBackOff) {
				return lserrors.NewError("RunInitContainer", initStatus.State.Waiting.Reason, initStatus.State.Waiting.Message)
			}
		}
	}

	waitStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.WaitContainerName)
	if err == nil {
		if waitStatus.State.Waiting != nil {
			if utils.StringIsOneOf(waitStatus.State.Waiting.Reason,
				kutil.ErrImagePull,
				kutil.ErrInvalidImageName,
				kutil.ErrRegistryUnavailable,
				kutil.ErrImageNeverPull,
				kutil.ErrImagePullBackOff) {
				return lserrors.NewError("RunWaitContainer", waitStatus.State.Waiting.Reason, waitStatus.State.Waiting.Message)
			}
		}
	}

	mainStatus, err := kutil.GetStatusForContainer(pod.Status.ContainerStatuses, container.MainContainerName)
	if err == nil {
		if mainStatus.State.Waiting != nil {
			if utils.StringIsOneOf(mainStatus.State.Waiting.Reason,
				kutil.ErrImagePull,
				kutil.ErrInvalidImageName,
				kutil.ErrRegistryUnavailable,
				kutil.ErrImageNeverPull,
				kutil.ErrImagePullBackOff) {
				return lserrors.NewError("RunMainContainer", mainStatus.State.Waiting.Reason, mainStatus.State.Waiting.Message)
			}
		}
	}

	return nil
}

// to find a suitable secret for images on Docker Hub, we need its two domains to do matching
const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"
)

// syncSecrets identifies a authSecrets based on a given registry. It then creates or updates a pull secret in a target for the matched authSecret.
func (c *Container) syncSecrets(ctx context.Context,
	secretName,
	imageReference string,
	keyring *credentials.GeneralOciKeyring,
	defaultLabels map[string]string) (string, error) {
	authConfig := keyring.Get(imageReference)
	if authConfig == nil {
		return "", nil
	}

	imageref, err := dockerreference.ParseDockerRef(imageReference)
	if err != nil {
		return "", fmt.Errorf("not a valid imageReference reference %s: %w", imageReference, err)
	}

	host := dockerreference.Domain(imageref)
	// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
	if host == dockerHubDomain {
		host = dockerHubLegacyDomain
	}

	targetAuthConfigs := &dockerconfigfile.ConfigFile{
		AuthConfigs: map[string]types.AuthConfig{
			host: {
				Username:      authConfig.GetUsername(),
				Password:      authConfig.GetPassword(),
				Auth:          authConfig.GetAuth(),
				IdentityToken: authConfig.GetIdentityToken(),
				RegistryToken: authConfig.GetRegistryToken(),
			},
		},
	}

	targetAuthConfigsJSON, err := json.Marshal(targetAuthConfigs)
	if err != nil {
		return "", fmt.Errorf("failed to create valid auth config JSON for %s: %w", host, err)
	}

	authSecret := &corev1.Secret{}
	authSecret.Name = secretName
	authSecret.Namespace = c.Configuration.Namespace
	authSecret.Type = corev1.SecretTypeDockerConfigJson
	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostUncachedClient, authSecret, func() error {
		InjectDefaultLabels(authSecret, defaultLabels)
		kutil.SetMetaDataLabel(&authSecret.ObjectMeta, container.ContainerDeployerTypeLabel, "registry-pull-secret")
		authSecret.Data = map[string][]byte{
			corev1.DockerConfigJsonKey: targetAuthConfigsJSON,
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("unable to image pull secret to host cluster: %w", err)
	}
	return authSecret.Name, nil
}

// parseAndSyncSecrets parses and synchronizes relevant pull secrets for container image, blueprint & component descriptor secrets from the landscaper and host cluster.
func (c *Container) parseAndSyncSecrets(ctx context.Context, defaultLabels map[string]string) (imagePullSecret, blueprintSecret, componentDescriptorSecret string, erro error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	fs := osfs.New()
	// find the secrets that match our image, our blueprint and our componentdescriptor
	ociKeyring := credentials.New()

	if c.Configuration.OCI != nil {
		for _, secretFileName := range c.Configuration.OCI.ConfigFiles {
			secretFileContent, err := vfs.ReadFile(fs, secretFileName)
			if err != nil {
				log.Debug("Unable to read auth config from file, skipping", lc.KeyFileName, secretFileName, lc.KeyError, err.Error())
				continue
			}

			authConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(secretFileContent))
			if err != nil {
				log.Debug("Invalid auth config in secret, skipping", lc.KeyFileName, secretFileName, lc.KeyError, err.Error())
				continue
			}

			if err := ociKeyring.Add(authConfig.GetCredentialsStore("")); err != nil {
				erro = fmt.Errorf("unable to add config from %q to credentials store: %w", secretFileName, err)
				return
			}
		}
	}

	secretRefs := append(lib.GetRegistryPullSecretsFromContext(c.Context), c.ProviderConfiguration.RegistryPullSecrets...)
	for _, secretRef := range secretRefs {
		secret := &corev1.Secret{}

		err := c.lsUncachedClient.Get(ctx, secretRef.NamespacedName(), secret)
		if err != nil {
			log.Debug("Unable to get auth config from secret, skipping", lc.KeyResource, secretRef.NamespacedName().String(), lc.KeyError, err.Error())
		}
		authConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(secret.Data[corev1.DockerConfigJsonKey]))
		if err != nil {
			log.Debug("Invalid auth config in secret, skipping", lc.KeyResource, secretRef.NamespacedName().String(), lc.KeyError, err.Error())
			continue
		}
		if err := ociKeyring.Add(authConfig.GetCredentialsStore("")); err != nil {
			erro = fmt.Errorf("unable to add config from secret %q to credentials store: %w", secretRef.NamespacedName().String(), err)
			return
		}
	}

	var err error
	imagePullSecret, err = c.syncSecrets(ctx, ImagePullSecretName(c.DeployItem.Namespace, c.DeployItem.Name), c.ProviderConfiguration.Image, ociKeyring, defaultLabels)
	if err != nil {
		erro = fmt.Errorf("unable to obtain and sync image pull secret to host cluster: %w", err)
		return
	}

	// sync pull secrets for Component Descriptor
	// sync pull secrets for oci registry repositories
	if c.ProviderConfiguration.ComponentDescriptor != nil &&
		c.ProviderConfiguration.ComponentDescriptor.Reference != nil &&
		c.ProviderConfiguration.ComponentDescriptor.Reference.RepositoryContext != nil &&
		c.ProviderConfiguration.ComponentDescriptor.Reference.RepositoryContext.GetType() == cdv2.OCIRegistryType {

		ociRepoCtx := cdv2.OCIRegistryRepository{}
		if err := c.ProviderConfiguration.ComponentDescriptor.Reference.RepositoryContext.DecodeInto(&ociRepoCtx); err != nil {
			erro = fmt.Errorf("unable to decode oci repository context: %w", err)
			return
		}
		cdRef, err := cdoci.OCIRef(ociRepoCtx, c.ProviderConfiguration.ComponentDescriptor.Reference.ComponentName, c.ProviderConfiguration.ComponentDescriptor.Reference.Version)
		if err != nil {
			erro = fmt.Errorf("unable to generate component descriptor oci reference: %w", err)
			return
		}
		componentDescriptorSecret, err = c.syncSecrets(ctx, ComponentDescriptorPullSecretName(c.DeployItem.Namespace, c.DeployItem.Name), cdRef, ociKeyring, defaultLabels)
		if err != nil {
			erro = fmt.Errorf("unable to obtain and sync component descriptor secret to host cluster: %w", err)
			return
		}
	}

	// sync pull secrets for BluePrint and ocm config
	if c.ProviderConfiguration.Blueprint != nil && c.ProviderConfiguration.Blueprint.Reference != nil && c.ProviderConfiguration.ComponentDescriptor != nil {
		var ocmConfig *corev1.ConfigMap
		if c.Context.OCMConfig != nil {
			ocmConfig = &corev1.ConfigMap{}
			if err := c.lsUncachedClient.Get(ctx, client.ObjectKey{
				Namespace: c.Context.Namespace,
				Name:      c.Context.OCMConfig.Name,
			}, ocmConfig); err != nil {
				log.Debug("unable to get ocm config from config map", lc.KeyResource, fmt.Sprintf("%s/%s", ocmConfig.GetNamespace(), ocmConfig.GetName()), lc.KeyError, err.Error())
				erro = fmt.Errorf("unable to get ocm config from config map: %w", err)
				return
			}
		}

		registryAccess, err := registries.GetFactory(c.Context.UseOCM).NewRegistryAccess(ctx, fs, ocmConfig, nil, c.sharedCache, nil, c.Configuration.OCI, c.ProviderConfiguration.ComponentDescriptor.Inline)
		if err != nil {
			erro = fmt.Errorf("unable create registry reference to resolve component descriptor for ref %#v: %w", c.ProviderConfiguration.Blueprint.Reference, err)
			return
		}

		compRef := deployerlegacy.GetReferenceFromComponentDescriptorDefinition(c.ProviderConfiguration.ComponentDescriptor)
		blueprintName := c.ProviderConfiguration.Blueprint.Reference.ResourceName

		componentVersion, err := registryAccess.GetComponentVersion(ctx, compRef)
		if err != nil {
			erro = fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", c.ProviderConfiguration.Blueprint.Reference, err)
			return
		}

		resource, err := blueprints.GetBlueprintResourceFromComponentVersion(componentVersion, blueprintName)
		if err != nil {
			erro = fmt.Errorf("unable to find blueprint resource in component descriptor for ref %#v: %w", c.ProviderConfiguration.Blueprint.Reference, err)
			return
		}

		// currently only 2 access methods are supported: localOCIBlob and oci artifact
		// if the resource is a local oci blob then the same credentials as for the component descriptor is used
		if resource.GetAccessType() == cdv2.LocalOCIBlobType {
			return
		}

		// if the resource is an oci artifact then we need to parse the actual oci image reference
		ociRegistryAccess := &cdv2.OCIRegistryAccess{}
		resourceEntry, err := resource.GetResource()
		if err != nil {
			erro = fmt.Errorf("unable to get entry of the blueprint resource in component descriptor for ref %#v: %w", c.ProviderConfiguration.Blueprint.Reference, err)
			return
		}

		accessRaw := resourceEntry.Access.Raw
		if err := cdv2.NewCodec(nil, nil, nil).Decode(accessRaw, ociRegistryAccess); err != nil {
			erro = fmt.Errorf("unable to parse oci registry access of blueprint resource in component descriptor for ref %#v: %w", c.ProviderConfiguration.Blueprint.Reference, err)
			return
		}

		blueprintSecret, err = c.syncSecrets(ctx, BluePrintPullSecretName(c.DeployItem.Namespace, c.DeployItem.Name), ociRegistryAccess.ImageReference, ociKeyring, defaultLabels)
		if err != nil {
			erro = fmt.Errorf("unable to obtain and sync blueprint pull secret to host cluster: %w", err)
			return
		}
	}

	return
}

// SyncConfiguration syncs the provider configuration data as secret to the host cluster.
func (c *Container) SyncConfiguration(ctx context.Context, defaultLabels map[string]string) error {
	secret := &corev1.Secret{}
	secret.Name = ConfigurationSecretName(c.DeployItem.Namespace, c.DeployItem.Name)
	secret.Namespace = c.Configuration.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostUncachedClient, secret, func() error {
		InjectDefaultLabels(secret, defaultLabels)
		kutil.SetMetaDataLabel(&secret.ObjectMeta, container.ContainerDeployerTypeLabel, "configuration")
		secret.Data = map[string][]byte{
			container.ConfigurationFilename: c.DeployItem.Spec.Configuration.Raw,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to sync provider configuration to host cluster: %w", err)
	}
	return nil
}

func (c *Container) SyncOCMConfiguration(ctx context.Context, defaultLabels map[string]string) error {
	configmap := &corev1.ConfigMap{}
	configmap.Data = map[string]string{}
	configmap.Name = OCMConfigConfigMapName(c.DeployItem.Namespace, c.DeployItem.Name)
	configmap.Namespace = c.Configuration.Namespace

	if c.Context.OCMConfig != nil {
		ocmConfig := corev1.ConfigMap{}
		if err := c.lsUncachedClient.Get(ctx, client.ObjectKey{
			Namespace: c.Context.Namespace,
			Name:      c.Context.OCMConfig.Name,
		}, &ocmConfig); err != nil {
			return err
		}
		configmap.Data = ocmConfig.Data
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostUncachedClient, configmap, func() error {
		InjectDefaultLabels(configmap, defaultLabels)
		kutil.SetMetaDataLabel(&configmap.ObjectMeta, container.ContainerDeployerTypeLabel, "configuration")
		return nil
	}); err != nil {
		return fmt.Errorf("unable to sync ocm configuration to host cluster: %w", err)
	}
	return nil
}

// SyncTarget syncs the deployitem's target content as secret to the host cluster.
func (c *Container) SyncTarget(ctx context.Context, defaultLabels map[string]string) error {
	secret := &corev1.Secret{}
	secret.Name = TargetSecretName(c.DeployItem.Namespace, c.DeployItem.Name)
	secret.Namespace = c.Configuration.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.hostUncachedClient, secret, func() error {
		InjectDefaultLabels(secret, defaultLabels)
		kutil.SetMetaDataLabel(&secret.ObjectMeta, container.ContainerDeployerTypeLabel, "target")
		data, err := json.Marshal(c.Target)
		if err != nil {
			return fmt.Errorf("error marshalling resolvedtarget struct into json: %w", err)
		}
		secret.Data = map[string][]byte{
			container.TargetFileName: data,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to sync target content to host cluster: %w", err)
	}
	return nil
}

// SyncExport syncs the export secret from the wait container to the deploy item export.
func (c *Container) SyncExport(ctx context.Context) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Debug("Sync export to landscaper cluster")
	secret := &corev1.Secret{}
	key := kutil.ObjectKey(ExportSecretName(c.DeployItem.Namespace, c.DeployItem.Name), c.Configuration.Namespace)
	if err := c.hostUncachedClient.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("No export found for deploy item", "deployitem", key.String())
			return nil
		}
		return fmt.Errorf("unable to fetch exported secret %s from host cluster: %w", ExportSecretName(c.DeployItem.Namespace, c.DeployItem.Name), err)
	}

	expSecret := &corev1.Secret{}
	expSecret.Name = DeployItemExportSecretName(c.DeployItem.Name)
	expSecret.Namespace = c.DeployItem.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, c.lsUncachedClient, expSecret, func() error {
		expSecret.Data = secret.Data
		return controllerutil.SetControllerReference(c.DeployItem, expSecret, api.LandscaperScheme)
	}); err != nil {
		return fmt.Errorf("unable to sync export to landscaper cluster: %w", err)
	}

	c.DeployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      expSecret.Name,
		Namespace: expSecret.Namespace,
	}

	return nil
}

// CleanupPod cleans up a pod that was started with the container deployer.
func (c *Container) CleanupPod(ctx context.Context, pod *corev1.Pod) error {
	return CleanupPod(ctx, c.hostUncachedClient, pod, c.Configuration.DebugOptions != nil && c.Configuration.DebugOptions.KeepPod)
}
