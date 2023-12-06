// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/apis/deployer/container"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type GarbageCollector struct {
	log           logging.Logger
	deployerID    string
	hostNamespace string
	config        containerv1alpha1.GarbageCollection
	lsClient      client.Client
	hostClient    client.Client
	requeueAfter  time.Duration
	keepPods      bool
}

// NewGarbageCollector creates a new Garbage collector that cleanups leaked service accounts, rbac rules and pods.
func NewGarbageCollector(log logging.Logger,
	lsClient,
	hostClient client.Client,
	deployerID,
	hostNamespace string,
	config containerv1alpha1.GarbageCollection,
	keepPods bool) *GarbageCollector {
	return &GarbageCollector{
		log:           log,
		deployerID:    deployerID,
		hostNamespace: hostNamespace,
		config:        config,
		lsClient:      lsClient,
		hostClient:    hostClient,
		requeueAfter:  time.Duration(config.RequeueTimeSeconds) * time.Second,
		keepPods:      keepPods,
	}
}

func (gc *GarbageCollector) StartDeployerJob(ctx context.Context) error {
	gc.log.Info("GarbageCollector: starting garbage collection")

	wait.UntilWithContext(ctx, gc.Cleanup, gc.requeueAfter)
	return nil
}

func (gc *GarbageCollector) Cleanup(ctx context.Context) {
	ctx = logging.NewContext(ctx, gc.log)
	logger, ctx := logging.FromContextOrNew(ctx, nil)

	listOptions := []client.ListOption{client.InNamespace(gc.hostNamespace),
		client.HasLabels{container.ContainerDeployerDeployItemNameLabel, container.ContainerDeployerDeployItemNamespaceLabel}}
	if len(gc.deployerID) != 0 {
		listOptions = []client.ListOption{client.InNamespace(gc.hostNamespace),
			client.HasLabels{container.ContainerDeployerDeployItemNameLabel, container.ContainerDeployerDeployItemNamespaceLabel},
			client.MatchingLabels{container.ContainerDeployerIDLabel: gc.deployerID}}
	}

	// cleanup service accounts
	saList := &corev1.ServiceAccountList{}
	if err := gc.hostClient.List(ctx, saList, listOptions...); err != nil {
		logger.Error(err, err.Error())
	}

	for i := range saList.Items {
		next := &saList.Items[i]
		if err := gc.cleanupRBACResources(ctx, next); err != nil {
			logger.Error(err, "cleanup service account", lc.KeyResource, kutil.ObjectKeyFromObject(next).String())
		}
	}

	// cleanup roles
	roleList := &rbacv1.RoleList{}
	if err := gc.hostClient.List(ctx, roleList, listOptions...); err != nil {
		logger.Error(err, err.Error())
	}

	for i := range roleList.Items {
		next := &roleList.Items[i]
		if err := gc.cleanupRBACResources(ctx, next); err != nil {
			logger.Error(err, "cleanup role", lc.KeyResource, kutil.ObjectKeyFromObject(next).String())
		}
	}

	// cleanup rolesbidings
	roleBindingList := &rbacv1.RoleBindingList{}
	if err := gc.hostClient.List(ctx, roleBindingList, listOptions...); err != nil {
		logger.Error(err, err.Error())
	}

	for i := range roleBindingList.Items {
		next := &roleBindingList.Items[i]
		if err := gc.cleanupRBACResources(ctx, next); err != nil {
			logger.Error(err, "cleanup rolebinding", lc.KeyResource, kutil.ObjectKeyFromObject(next).String())
		}
	}

	// cleanup secrets
	secretList := &corev1.SecretList{}
	if err := read_write_layer.ListSecrets(ctx, gc.hostClient, secretList, read_write_layer.R000074, listOptions...); err != nil {
		logger.Error(err, err.Error())
	}

	for i := range secretList.Items {
		next := &secretList.Items[i]
		if err := gc.cleanupSecret(ctx, next); err != nil {
			logger.Error(err, "cleanup secret", lc.KeyResource, kutil.ObjectKeyFromObject(next).String())
		}
	}

	if !gc.keepPods {
		// cleanup pods
		podList := &corev1.PodList{}
		if err := read_write_layer.ListPods(ctx, gc.hostClient, podList, read_write_layer.R000075, listOptions...); err != nil {
			logger.Error(err, err.Error())
		}

		for i := range podList.Items {
			next := &podList.Items[i]
			if err := gc.cleanupPod(ctx, next); err != nil {
				logger.Error(err, "cleanup pod", lc.KeyResource, kutil.ObjectKeyFromObject(next).String())
			}
		}
	}
}

func (gc *GarbageCollector) cleanupRBACResources(ctx context.Context, obj client.Object) error {
	shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
	if err != nil {
		return err
	}
	if !shouldGC {
		return nil
	}

	di := &lsv1alpha1.DeployItem{}
	di.Name = obj.GetLabels()[container.ContainerDeployerDeployItemNameLabel]
	di.Namespace = obj.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]
	if err := CleanupRBAC(ctx, di, gc.hostClient, obj.GetNamespace()); err != nil {
		return err
	}
	return nil
}

// cleanupSecret deletes secrets that do not have a parent deploy item anymore.
func (gc *GarbageCollector) cleanupSecret(ctx context.Context, obj *corev1.Secret) error {
	shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
	if err != nil {
		return err
	}
	if !shouldGC {
		return nil
	}

	if err := gc.hostClient.Delete(ctx, obj); err != nil {
		return err
	}
	return nil
}

// cleanupPod deletes pods that do not have a parent deploy item anymore.
func (gc *GarbageCollector) cleanupPod(ctx context.Context, obj *corev1.Pod) error {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	if obj.Status.Phase == corev1.PodPending || obj.Status.Phase == corev1.PodRunning || obj.Status.Phase == corev1.PodUnknown {
		logger.Debug("Not garbage collected", lc.KeyReason, "pod is still running", lc.KeyPhase, obj.Status.Phase)
		return nil
	}

	shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
	if err != nil {
		return err
	}
	if shouldGC {
		// always garbage collect pods that do not have a corresponding deployitem anymore
		logger.Debug("Garbage collected", lc.KeyReason, "deploy item does not exist anymore")
		if err := CleanupPod(ctx, gc.hostClient, obj, false); err != nil {
			return fmt.Errorf("unable to garbage collect pod %s: %w", kutil.ObjectKeyFromObject(obj).String(), err)
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(obj, container.ContainerDeployerFinalizer) {
		logger.Debug("Garbage collected", lc.KeyReason, "pod has no finalizer")
		err := gc.hostClient.Delete(ctx, obj)
		return err
	}

	isLatest, err := gc.isLatestPod(ctx, obj)
	if err != nil {
		return err
	}
	if isLatest {
		logger.Debug("Not garbage collected", lc.KeyReason, "latest pod")
		return nil
	}

	if err := CleanupPod(ctx, gc.hostClient, obj, false); err != nil {
		return fmt.Errorf("unable to garbage collect pod %s: %w", kutil.ObjectKeyFromObject(obj).String(), err)
	}
	logger.Debug("Garbage collected")
	return nil
}

// isLatestPod cleans returns if the current pod is the latest executed pod.
func (gc *GarbageCollector) isLatestPod(ctx context.Context, pod *corev1.Pod) (bool, error) {
	var (
		diName      = pod.Labels[container.ContainerDeployerDeployItemNameLabel]
		diNamespace = pod.Labels[container.ContainerDeployerDeployItemNamespaceLabel]
	)

	podList := &corev1.PodList{}
	if err := read_write_layer.ListPods(ctx, gc.hostClient, podList, read_write_layer.R000076,
		client.InNamespace(gc.hostNamespace),
		client.MatchingLabels{
			container.ContainerDeployerDeployItemNameLabel:      diName,
			container.ContainerDeployerDeployItemNamespaceLabel: diNamespace,
		}); err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		return false, fmt.Errorf("no pods found in the host namespace %s", gc.hostNamespace)
	}

	// only return latest pod and ignore previous runs
	var latest *corev1.Pod
	for _, p := range podList.Items {
		// ignore pods with no finalizer as they are already reconciled and their state was persisted.
		if !controllerutil.ContainsFinalizer(&p, container.ContainerDeployerFinalizer) {
			continue
		}
		if latest == nil {
			latest = p.DeepCopy()
			continue
		}
		if p.CreationTimestamp.Equal(&latest.CreationTimestamp) {
			// currently only for test debugging.
			// remove as soon as the test is stable.
			gc.log.Debug("Creation time equals", "currentPod", p.Name, "latest", latest.Name)
		}
		if p.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = p.DeepCopy()
		}
	}
	if latest == nil {
		return false, nil
	}

	return latest.Name == pod.Name, nil // namespace is irrelevant
}

// shouldGarbageCollect checks whether the object should be garbage collected.
// By default, an object should be garbage collected if the corresponding deploy item has been deleted.
func (gc *GarbageCollector) shouldGarbageCollect(ctx context.Context, obj client.Object) (bool, error) {
	di := &lsv1alpha1.DeployItem{}
	key := types.NamespacedName{
		Namespace: obj.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel],
		Name:      obj.GetLabels()[container.ContainerDeployerDeployItemNameLabel],
	}
	logger := gc.log.WithValues("deployItem", key.String(), lc.KeyResource, kutil.ObjectKeyFromObject(obj).String())
	if err := read_write_layer.GetDeployItem(ctx, gc.lsClient, key, di, read_write_layer.R000036); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		// do not cleanup as we are unsure about the state of the deploy item.
		return false, err
	}
	logger.Debug("DeployItem still exists, resource should not be garbage collected")
	return false, nil
}
