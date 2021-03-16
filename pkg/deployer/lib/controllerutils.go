// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package lib

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ConstructReconcileRequestFunc describes a function that is used to transform a pod into a
// reconcile request.
// Can return false if no request should be triggered.
type ConstructReconcileRequestFunc = func(obj metav1.Object) (reconcile.Request, bool)

// PodEventHandler implements the controller runtime handler interface
// that reconciles only pods that are created by this controller
type PodEventHandler struct {
	constructRequest ConstructReconcileRequestFunc
}

// NewPodEventHandler creates a new pod event handler.
func NewPodEventHandler(f ConstructReconcileRequestFunc) *PodEventHandler {
	return &PodEventHandler{
		constructRequest: f,
	}
}

var _ handler.EventHandler = &PodEventHandler{}

func (p *PodEventHandler) Create(event event.CreateEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.constructRequest(event.Object); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Update(event event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.constructRequest(event.ObjectNew); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Delete(event event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.constructRequest(event.Object); ok {
		q.Add(req)
	}
}

func (p *PodEventHandler) Generic(event event.GenericEvent, q workqueue.RateLimitingInterface) {
	if req, ok := p.constructRequest(event.Object); ok {
		q.Add(req)
	}
}

// NoopPredicate is a predicate definition that does not react on any event.
// its used for the pod reconciler that should only be triggered upon pod events not deploy item events
type NoopPredicate struct{}

func (n NoopPredicate) Create(createEvent event.CreateEvent) bool {
	return false
}

func (n NoopPredicate) Delete(deleteEvent event.DeleteEvent) bool {
	return false
}

func (n NoopPredicate) Update(updateEvent event.UpdateEvent) bool {
	return false
}

func (n NoopPredicate) Generic(genericEvent event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = NoopPredicate{}
