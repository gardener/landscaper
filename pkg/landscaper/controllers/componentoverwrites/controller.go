// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// NewController creates a new component overwrite controller.
func NewController(log logging.Logger, c client.Client, scheme *runtime.Scheme, mgr *componentoverwrites.Manager) reconcile.Reconciler {
	return &controller{log: log, client: c, scheme: scheme, mgr: mgr}
}

type controller struct {
	log    logging.Logger
	client client.Client
	scheme *runtime.Scheme
	mgr    *componentoverwrites.Manager
}

func (con *controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := con.log.WithValues("resource", req.NamespacedName.String())

	co := &lsv1alpha1.ComponentOverwrites{}
	if err := con.client.Get(ctx, req.NamespacedName, co); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Logr().V(5).Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	for _, overwrite := range co.Overwrites {
		con.mgr.Add(overwrite)
	}
	return reconcile.Result{}, nil
}
