// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"

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
}

// NewGarbageCollector creates a new Garbage collector that cleanups leaked service accounts, rbac rules and pods.
func NewGarbageCollector(log logging.Logger,
	lsClient,
	hostClient client.Client,
	deployerID,
	hostNamespace string,
	config containerv1alpha1.GarbageCollection) *GarbageCollector {
	return &GarbageCollector{
		log:           log,
		deployerID:    deployerID,
		hostNamespace: hostNamespace,
		config:        config,
		lsClient:      lsClient,
		hostClient:    hostClient,
		requeueAfter:  time.Duration(config.RequeueTimeSeconds) * time.Second,
	}
}

// Add configures the watches for the resources that should be garbage collected.
func (gc *GarbageCollector) Add(hostMgr manager.Manager, keepPods bool) error {
	pred := &ManagedResourcesPredicate{
		Log:           gc.log,
		DeployerID:    gc.deployerID,
		HostNamespace: gc.hostNamespace,
	}

	objectsToClean := []client.Object{
		&corev1.ServiceAccount{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}

	for _, obj := range objectsToClean {
		err := ctrl.NewControllerManagedBy(hostMgr).
			For(obj, builder.WithPredicates(pred)).
			WithOptions(controller.Options{
				MaxConcurrentReconciles: gc.config.Worker,
			}).
			Complete(gc.cleanupRBACResources(obj.DeepCopyObject().(client.Object)))
		if err != nil {
			return err
		}
		gc.log.Info("registered container garbage collector", "resource", reflect.TypeOf(obj).String())
	}

	err := ctrl.NewControllerManagedBy(hostMgr).
		For(&corev1.Secret{}, builder.WithPredicates(pred)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: gc.config.Worker,
		}).
		Complete(reconcile.Func(gc.cleanupSecret))
	if err != nil {
		return err
	}
	gc.log.Info("registered container garbage collector", "resource", "Secrets")

	if !keepPods {
		err = ctrl.NewControllerManagedBy(hostMgr).
			For(&corev1.Pod{}, builder.WithPredicates(pred)).
			WithOptions(controller.Options{
				MaxConcurrentReconciles: gc.config.Worker,
			}).
			Complete(reconcile.Func(gc.cleanupPod))
		if err != nil {
			return err
		}
		gc.log.Info("registered container garbage collector", "resource", "Pods")
	}

	return nil
}

func (gc *GarbageCollector) cleanupRBACResources(obj client.Object) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		obj := obj.DeepCopyObject().(client.Object)
		if err := gc.hostClient.Get(ctx, req.NamespacedName, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			gc.log.Error(err, "unable to get resource")
			return reconcile.Result{}, err
		}
		shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !shouldGC {
			return reconcile.Result{Requeue: true, RequeueAfter: gc.requeueAfter}, nil
		}

		di := &lsv1alpha1.DeployItem{}
		di.Name = obj.GetLabels()[container.ContainerDeployerDeployItemNameLabel]
		di.Namespace = obj.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]
		if err := CleanupRBAC(ctx, di, gc.hostClient, obj.GetNamespace()); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	})
}

// cleanupSecret deletes secrets that do not have a parent deploy item anymore.
func (gc *GarbageCollector) cleanupSecret(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	obj := &corev1.Secret{}
	if err := gc.hostClient.Get(ctx, req.NamespacedName, obj); err != nil {
		return reconcile.Result{}, err
	}
	shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !shouldGC {
		return reconcile.Result{Requeue: true, RequeueAfter: gc.requeueAfter}, nil
	}

	if err := gc.hostClient.Delete(ctx, obj); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to garbage collect secret %s: %w", kutil.ObjectKeyFromObject(obj).String(), err)
	}
	return reconcile.Result{}, nil
}

// cleanupPod deletes pods that do not have a parent deploy item anymore.
func (gc *GarbageCollector) cleanupPod(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	obj := &corev1.Pod{}
	if err := gc.hostClient.Get(ctx, req.NamespacedName, obj); err != nil {
		return reconcile.Result{}, err
	}
	logger := gc.log.WithValues("type", "Pod", "resource", kutil.ObjectKeyFromObject(obj).String())

	if obj.Status.Phase == corev1.PodPending || obj.Status.Phase == corev1.PodRunning || obj.Status.Phase == corev1.PodUnknown {
		logger.Logr().V(9).Info("not garbage collected", "reason", "pod is still running", "phase", obj.Status.Phase)
		return reconcile.Result{}, nil
	}

	shouldGC, err := gc.shouldGarbageCollect(ctx, obj)
	if err != nil {
		return reconcile.Result{}, err
	}
	if shouldGC {
		// always garbage collect pods that do not have a corresponding deployitem anymore
		logger.Logr().V(10).Info("garbage collected", "reason", "deploy item does not exist anymore")
		if err := CleanupPod(ctx, gc.hostClient, obj, false); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to garbage collect pod %s: %w", kutil.ObjectKeyFromObject(obj).String(), err)
		}
		return reconcile.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(obj, container.ContainerDeployerFinalizer) {
		logger.Logr().V(9).Info("garbage collected", "reason", "pod has no finalizer")
		err := gc.hostClient.Delete(ctx, obj)
		return reconcile.Result{}, err
	}

	isLatest, err := gc.isLatestPod(ctx, obj)
	if err != nil {
		return reconcile.Result{}, err
	}
	if isLatest {
		logger.Logr().V(9).Info("not garbage collected", "reason", "latest pod")
		return reconcile.Result{Requeue: true, RequeueAfter: gc.requeueAfter}, nil
	}

	if err := CleanupPod(ctx, gc.hostClient, obj, false); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to garbage collect pod %s: %w", kutil.ObjectKeyFromObject(obj).String(), err)
	}
	logger.Logr().V(9).Info("garbage collected")
	return reconcile.Result{}, nil
}

// isLatestPod cleans returns if the current pod is the latest executed pod.
func (gc *GarbageCollector) isLatestPod(ctx context.Context, pod *corev1.Pod) (bool, error) {
	var (
		diName      = pod.Labels[container.ContainerDeployerDeployItemNameLabel]
		diNamespace = pod.Labels[container.ContainerDeployerDeployItemNamespaceLabel]
	)

	podList := &corev1.PodList{}
	if err := gc.hostClient.List(ctx, podList, client.InNamespace(gc.hostNamespace), client.MatchingLabels{
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
			gc.log.Logr().V(12).Info("creation time equals", "currentPod", p.Name, "latest", latest.Name)
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
	logger := gc.log.WithValues("deployItem", key.String(), "resource", kutil.ObjectKeyFromObject(obj).String())
	if err := read_write_layer.GetDeployItem(ctx, gc.lsClient, key, di); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		// do not cleanup as we are unsure about the state of the deploy item.
		return false, err
	}
	logger.Logr().V(9).Info("DeployItem still exists")
	return false, nil
}

// ManagedResourcesPredicate implements the controller runtime handler interface
// that reconciles resources managed by the deployer.
type ManagedResourcesPredicate struct {
	Log           logging.Logger
	DeployerID    string
	HostNamespace string
}

var _ predicate.Predicate = &ManagedResourcesPredicate{}

func (h *ManagedResourcesPredicate) shouldReconcile(object metav1.Object) bool {
	logger := h.Log.WithValues("resource", kutil.ObjectKeyFromObject(object).String())
	if object.GetNamespace() != h.HostNamespace {
		logger.Logr().V(10).Info("not garbage collected", "reason", "namespace does not match", "hostNamespace", h.HostNamespace, "resourceNamespace", object.GetNamespace())
		return false
	}
	if _, ok := object.GetLabels()[container.ContainerDeployerDeployItemNameLabel]; !ok {
		logger.Logr().V(9).Info("not garbage collected", "reason", "no deploy item name label")
		return false
	}
	if _, ok := object.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]; !ok {
		logger.Logr().V(9).Info("not garbage collected", "reason", "no deploy item namespace label")
		return false
	}

	if len(h.DeployerID) != 0 {
		if id, ok := object.GetLabels()[container.ContainerDeployerIDLabel]; !ok || id != h.DeployerID {
			logger.Logr().V(9).Info("not garbage collected", "reason", "deployer ids do not match")
			return false
		}
	}
	logger.Logr().V(10).Info("enqueue for garbage collection")
	return true
}

func (h *ManagedResourcesPredicate) Create(event event.CreateEvent) bool {
	return h.shouldReconcile(event.Object)
}

func (h *ManagedResourcesPredicate) Update(event event.UpdateEvent) bool {
	return h.shouldReconcile(event.ObjectNew)
}

func (h *ManagedResourcesPredicate) Delete(event event.DeleteEvent) bool {
	return h.shouldReconcile(event.Object)
}

func (h *ManagedResourcesPredicate) Generic(event event.GenericEvent) bool {
	return h.shouldReconcile(event.Object)
}
