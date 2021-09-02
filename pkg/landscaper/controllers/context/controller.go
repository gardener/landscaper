// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// NewDefaulterController creates a new context controller that reconciles the default context object in the namespaces.
func NewDefaulterController(log logr.Logger,
	kubeClient client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	config *config.LandscaperConfiguration) (reconcile.Reconciler, error) {
	return &defaulterController{
		log:           log,
		client:        kubeClient,
		scheme:        scheme,
		eventRecorder: eventRecorder,
		config:        config,
	}, nil
}

type defaulterController struct {
	log           logr.Logger
	client        client.Client
	eventRecorder record.EventRecorder
	scheme        *runtime.Scheme
	config        *config.LandscaperConfiguration
}

func (c *defaulterController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := c.log.WithValues("resource", req.NamespacedName)
	logger.V(7).Info("reconcile")

	ns := &corev1.Namespace{}
	if err := c.client.Get(ctx, req.NamespacedName, ns); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defaultCtx := &lsv1alpha1.Context{}
	mut := func(lCtx *lsv1alpha1.Context) {
		lCtx.Name = lsv1alpha1.DefaultContextName
		lCtx.Namespace = ns.Name

		if c.config.RepositoryContext != nil {
			lCtx.RepositoryContext = c.config.RepositoryContext
		}
	}
	err := c.client.Get(ctx, kutil.ObjectKey(lsv1alpha1.DefaultContextName, ns.Name), defaultCtx)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		// create new context
		mut(defaultCtx)
		if err := c.client.Create(ctx, defaultCtx); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	old := defaultCtx.DeepCopy()
	mut(defaultCtx)
	if err := c.client.Patch(ctx, defaultCtx, client.MergeFrom(old)); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
