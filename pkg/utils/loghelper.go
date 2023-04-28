package utils

import (
	"context"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

type LogHelper struct {
}

func (r LogHelper) LogErrorAndGetReconcileResult(ctx context.Context, lsError lserrors.LsError) (reconcile.Result, error) {

	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "LogErrorAndGetReconcileResult")

	if lsError == nil {
		return reconcile.Result{}, nil
	} else if lserrors.ContainsErrorCode(lsError, lsv1alpha1.ErrorForInfoOnly) {
		logger.Info(lsError.Error())
		return reconcile.Result{Requeue: true}, nil
	} else {
		logger.Error(lsError, lsError.Error())
		return reconcile.Result{Requeue: true}, nil
	}
}
