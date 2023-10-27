package timeout

import (
	"context"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
)

// ignoreTimeoutChecker is a TimeoutChecker which never raises a timeout error.
type ignoreTimeoutChecker struct{}

func newIgnoreTimeoutChecker() *ignoreTimeoutChecker {
	return &ignoreTimeoutChecker{}
}

func (t *ignoreTimeoutChecker) TimeoutExceeded(_ context.Context, _ *lsv1alpha1.DeployItem, _ string) (time.Duration, lserrors.LsError) {
	return defaultTimeout, nil
}
