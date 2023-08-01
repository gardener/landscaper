// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// PodReconciler implements the reconciler.Reconcile interface that is expected to be called on
// pod events as described by the PodEventHandler.
// The reconciler basically calls the container reconcile.
type PodReconciler struct {
	log             logging.Logger
	lsClient        client.Client
	lsEventRecorder record.EventRecorder
	hostClient      client.Client
	config          containerv1alpha1.Configuration
	diRec           deployerlib.Deployer
	deployerType    lsv1alpha1.DeployItemType
	callerName      string
	targetSelectors []lsv1alpha1.TargetSelector
}

func NewPodReconciler(
	log logging.Logger,
	lsClient,
	hostClient client.Client,
	lsEventRecorder record.EventRecorder,
	config containerv1alpha1.Configuration,
	deployer deployerlib.Deployer,
	deployerType lsv1alpha1.DeployItemType,
	callerName string,
	targetSelectors []lsv1alpha1.TargetSelector) *PodReconciler {

	return &PodReconciler{
		log:             log,
		config:          config,
		lsClient:        lsClient,
		lsEventRecorder: lsEventRecorder,
		hostClient:      hostClient,
		diRec:           deployer,
		deployerType:    deployerType,
		callerName:      callerName,
		targetSelectors: targetSelectors,
	}
}

func (r *PodReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := r.log.StartReconcileAndAddToContext(ctx, req)

	if req.Namespace != lsutil.GetCurrentPodNamespace() {
		return reconcile.Result{}, nil
	}

	podMetadata := lsutil.EmptyPodMetadata()
	if err := r.lsClient.Get(ctx, req.NamespacedName, podMetadata); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	key, ok := r.getReconcileDeployItemRequest(podMetadata)
	if !ok {
		return reconcile.Result{}, nil
	}

	deployItem, err := getAndCheckReconcile(ctx, r.lsClient, r.config, key)
	if err != nil {
		return reconcile.Result{}, err
	}
	if deployItem == nil {
		return reconcile.Result{}, nil
	}

	if lock.LockerEnabled {
		diMetaData := lsutil.EmptyDeployItemMetadata()
		diMetaData.ObjectMeta = deployItem.ObjectMeta

		locker := lock.NewLocker(r.lsClient, r.hostClient, r.callerName)
		syncObject, lsErr := locker.LockDI(ctx, diMetaData)
		if lsErr != nil {
			return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, lsErr)
		}

		if syncObject == nil {
			return locker.NotLockedResult()
		}

		defer func() {
			locker.Unlock(ctx, syncObject)
		}()
	}

	lsv1alpha1helper.Touch(&deployItem.ObjectMeta)
	if err = read_write_layer.NewWriter(r.lsClient).UpdateDeployItem(ctx, read_write_layer.W000030, deployItem); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (p *PodReconciler) getReconcileDeployItemRequest(object metav1.Object) (client.ObjectKey, bool) {

	key := client.ObjectKey{}
	ok := false

	key.Name, ok = object.GetLabels()[container.ContainerDeployerDeployItemNameLabel]
	if !ok {
		return key, false
	}
	key.Namespace, ok = object.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]
	if !ok {
		return key, false
	}
	return key, true
}

type namespaceAndAnnotationPredicate struct {
	namespace string
}

var _ predicate.Predicate = &namespaceAndAnnotationPredicate{}

func newNamespaceAndAnnotationPredicate() *namespaceAndAnnotationPredicate {
	return &namespaceAndAnnotationPredicate{
		namespace: lsutil.GetCurrentPodNamespace(),
	}
}

func (n *namespaceAndAnnotationPredicate) Create(event event.CreateEvent) bool {
	return n.handleObject(event.Object)
}

func (n *namespaceAndAnnotationPredicate) Delete(event event.DeleteEvent) bool {
	return n.handleObject(event.Object)
}

func (n *namespaceAndAnnotationPredicate) Update(event event.UpdateEvent) bool {
	return n.handleObject(event.ObjectNew)
}

func (n *namespaceAndAnnotationPredicate) Generic(event event.GenericEvent) bool {
	return n.handleObject(event.Object)
}

func (n *namespaceAndAnnotationPredicate) handleObject(obj client.Object) bool {
	if obj.GetNamespace() != n.namespace {
		return false
	}

	if _, ok := obj.GetLabels()[container.ContainerDeployerDeployItemNameLabel]; !ok {
		return false
	}

	if _, ok := obj.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]; !ok {
		return false
	}

	return true
}
