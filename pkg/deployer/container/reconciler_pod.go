// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	lserror "github.com/gardener/landscaper/apis/errors"
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
}

func NewPodReconciler(
	log logging.Logger,
	lsClient,
	hostClient client.Client,
	lsEventRecorder record.EventRecorder,
	config containerv1alpha1.Configuration,
	deployer deployerlib.Deployer) *PodReconciler {
	return &PodReconciler{
		log:             log,
		config:          config,
		lsClient:        lsClient,
		lsEventRecorder: lsEventRecorder,
		hostClient:      hostClient,
		diRec:           deployer,
	}
}

func (r *PodReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	deployItem, lsCtx, err := GetAndCheckReconcile(r.log, r.lsClient, r.config)(ctx, req)
	if err != nil {
		return reconcile.Result{}, err
	}
	if deployItem == nil {
		return reconcile.Result{}, nil
	}
	old := deployItem.DeepCopy()
	err = r.diRec.Reconcile(ctx, lsCtx, deployItem, nil)
	lsErr := lserror.BuildLsErrorOrNil(err, "Reconcile", "Reconcile")
	return reconcile.Result{}, deployerlib.HandleErrorFunc(ctx, lsErr, r.lsClient, r.lsEventRecorder, old, deployItem, false)
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
