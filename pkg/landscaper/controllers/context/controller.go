// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	requeueImmediate = 10 * time.Millisecond
)

// NewDefaulterController creates a new context controller that reconciles the default context object in the namespaces.
func NewDefaulterController(lsUncachedClient, lsCachedClient client.Client,
	logger logging.Logger,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	config config.ContextControllerConfig) (reconcile.Reconciler, error) {
	return &defaulterController{
		lsUncachedClient:  lsUncachedClient,
		lsCachedClient:    lsCachedClient,
		log:               logger,
		scheme:            scheme,
		eventRecorder:     eventRecorder,
		config:            config,
		excludeNamespaces: sets.NewString(config.Default.ExcludedNamespaces...),
	}, nil
}

type defaulterController struct {
	lsUncachedClient  client.Client
	lsCachedClient    client.Client
	log               logging.Logger
	eventRecorder     record.EventRecorder
	scheme            *runtime.Scheme
	config            config.ContextControllerConfig
	excludeNamespaces sets.String //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
}

func (c *defaulterController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if c.excludeNamespaces.Has(req.Name) {
		return reconcile.Result{}, nil
	}

	logger := c.log.StartReconcile(req)
	ctx = logging.NewContext(ctx, logger)

	ns := &corev1.Namespace{}
	if err := c.lsUncachedClient.Get(ctx, req.NamespacedName, ns); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defaultCtx := &lsv1alpha1.Context{}
	defaultCtx.Name = lsv1alpha1.DefaultContextName
	defaultCtx.Namespace = ns.Name

	// we only want to overwrite the central managed configuration.
	// manual added configuration should be kept.
	if _, err := c.Writer().CreateOrPatchCoreContext(ctx, read_write_layer.W000077, defaultCtx, func() error {
		if c.config.Default.RepositoryContext != nil {
			defaultCtx.RepositoryContext = c.config.Default.RepositoryContext
		}
		return nil
	}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("default context not found", "err", err.Error())
		} else {
			logger.Error(err, "default context not created of patched")
		}

		return reconcile.Result{RequeueAfter: requeueImmediate}, nil
	}

	return reconcile.Result{}, nil
}

func (c *defaulterController) Writer() *read_write_layer.Writer {
	return read_write_layer.NewWriter(c.lsUncachedClient)
}
