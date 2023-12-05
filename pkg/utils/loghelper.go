package utils

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

type LogHelper struct {
}

func (LogHelper) LogErrorAndGetReconcileResult(ctx context.Context, lsError lserrors.LsError) (reconcile.Result, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "LogErrorAndGetReconcileResult")

	if lsError == nil {
		return reconcile.Result{}, nil
	} else if lserrors.ContainsErrorCode(lsError, lsv1alpha1.ErrorNoRetry) {
		logger.Info(lsError.Error())
		return reconcile.Result{Requeue: false}, nil
	} else if lserrors.ContainsErrorCode(lsError, lsv1alpha1.ErrorForInfoOnly) {
		logger.Info(lsError.Error())
		return reconcile.Result{Requeue: true}, nil
	} else {
		logger.Error(lsError, lsError.Error())
		return reconcile.Result{Requeue: true}, nil
	}
}

func (LogHelper) LogStandardErrorAndGetReconcileResult(ctx context.Context, err error) (reconcile.Result, error) {
	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "LogErrorAndGetReconcileResult")

	if err == nil {
		return reconcile.Result{}, nil
	}

	logger.Error(err, err.Error())
	return reconcile.Result{Requeue: true}, nil
}

func (LogHelper) LogErrorButNotFoundAsInfo(ctx context.Context, err error, message string) {
	logger, _ := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "LogErrorButNotFoundAsInfo")

	if err == nil {
		return
	} else if apierrors.IsNotFound(err) {
		logger.Info(message + ": " + err.Error())
	} else {
		logger.Error(err, message)
	}
}
