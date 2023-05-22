// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/lock"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
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

	metadata := lsutil.EmptyDeployItemMetadata()
	if err := r.lsClient.Get(ctx, req.NamespacedName, metadata); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// this check is only for compatibility reasons
	_, reponsible, lsErr := deployerlib.CheckResponsibility(ctx, r.lsClient, metadata, r.deployerType, r.targetSelectors)
	if lsErr != nil {
		return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, lsErr)
	}

	if !reponsible {
		return reconcile.Result{}, nil
	}

	locker := lock.NewLocker(r.lsClient, r.hostClient, r.callerName)
	syncObject, lsErr := locker.LockDI(ctx, metadata)
	if lsErr != nil {
		return lsutil.LogHelper{}.LogErrorAndGetReconcileResult(ctx, lsErr)
	}

	if syncObject == nil {
		return locker.NotLockedResult()
	}

	deployItem, _, err := GetAndCheckReconcile(r.lsClient, r.config)(ctx, req)
	if err != nil {
		return reconcile.Result{}, err
	}
	if deployItem == nil {
		return reconcile.Result{}, nil
	}

	lsv1alpha1helper.Touch(&deployItem.ObjectMeta)
	if err = read_write_layer.NewWriter(r.lsClient).UpdateDeployItem(ctx, read_write_layer.W000030, deployItem); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// PodEventHandler implements the controller runtime handler interface
// that reconciles only pods that are created by this controller
type PodEventHandler struct{}

var _ handler.EventHandler = &PodEventHandler{}

func (p *PodEventHandler) getReconcileDeployItemRequest(object metav1.Object) (reconcile.Request, bool) {
	var (
		req = reconcile.Request{}
		ok  bool
	)
	req.Name, ok = object.GetLabels()[container.ContainerDeployerDeployItemNameLabel]
	if !ok {
		return req, false
	}
	req.Namespace, ok = object.GetLabels()[container.ContainerDeployerDeployItemNamespaceLabel]
	if !ok {
		return req, false
	}
	return req, true
}

func (p *PodEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.getReconcileDeployItemRequest(event.Object); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.getReconcileDeployItemRequest(event.ObjectNew); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.getReconcileDeployItemRequest(event.Object); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.getReconcileDeployItemRequest(event.Object); ok {
		q.Add(req)
	}
}

// noopPredicate is a predicate definition that does not react on any event.
// its used for the pod reconciler that should only be triggered upon pod events not deploy item events
type noopPredicate struct{}

func (n noopPredicate) Create(createEvent event.CreateEvent) bool {
	return false
}

func (n noopPredicate) Delete(deleteEvent event.DeleteEvent) bool {
	return false
}

func (n noopPredicate) Update(updateEvent event.UpdateEvent) bool {
	return false
}

func (n noopPredicate) Generic(genericEvent event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = noopPredicate{}
