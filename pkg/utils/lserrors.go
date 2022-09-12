package utils

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
)

func IsRecoverableError(err error) bool {
	// currently there is only one intermediate error but this might change
	return lserrors.ContainsErrorCode(err, lsv1alpha1.ErrorWebhook)
}
