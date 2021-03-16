// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/deployer/container"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
)

// PodReconciler implements the reconciler.Reconcile interface that is expected to be called on
// pod events as described by the PodEventHandler.
// The reconciler basically calls the container reconcile.
type PodReconciler struct {
	log        logr.Logger
	lsClient   client.Client
	hostClient client.Client
	config     *containerv1alpha1.Configuration
	diRec      *DeployItemReconciler
}

func NewPodReconciler(log logr.Logger, lsClient, hostClient client.Client, config *containerv1alpha1.Configuration, diRec *DeployItemReconciler) *PodReconciler {
	return &PodReconciler{
		log:        log,
		config:     config,
		lsClient:   lsClient,
		hostClient: hostClient,
		diRec:      diRec,
	}
}

func (r *PodReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	deployItem, err := GetAndCheckReconcile(r.log, r.lsClient, r.config)(ctx, req)
	if err != nil {
		return reconcile.Result{}, err
	}
	if deployItem == nil {
		return reconcile.Result{}, nil
	}
	errHdl := deployerlib.HandleErrorFunc(r.log, r.lsClient, deployItem)
	if err := errHdl(ctx, r.diRec.reconcile(ctx, deployItem)); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// ConstructReconcileDeployItemRequest creates a reconcile request for a deploy item based on pod labels.
func ConstructReconcileDeployItemRequest(object metav1.Object) (reconcile.Request, bool) {
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
