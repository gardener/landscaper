package timeout

import (
	"context"
	"fmt"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

const (
	defaultTimeout = 10 * time.Minute
)

// standardTimeoutChecker is a TimeoutChecker which raises a timeout error if it takes more time to process a DeployItem
// than the default timeout, resp. the timeout specified in the DeployItem (spec.timeout).
// The processing time starts with the Init phase (status.transitionTimes.initTime) of the DeployItem.
type standardTimeoutChecker struct{}

func newStandardTimeoutChecker() *standardTimeoutChecker {
	return &standardTimeoutChecker{}
}

func (t *standardTimeoutChecker) TimeoutExceeded(ctx context.Context, deployItem *lsv1alpha1.DeployItem, checkpoint string) (time.Duration, lserrors.LsError) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	op := "StandardTimeoutChecker.TimeoutExceeded"

	if deployItem == nil {
		err := lserrors.NewError(op, checkpoint, "no deploy item provided")
		logger.Error(err, err.Error())
		return 0, err
	}

	if deployItem.Status.TransitionTimes == nil || deployItem.Status.TransitionTimes.InitTime == nil {
		err := lserrors.NewError(op, checkpoint, "status not initialized: transitionTimes.initTime is missing")
		logger.Error(err, err.Error())
		return 0, err
	}

	timeout := t.getTimeout(deployItem)

	currentTime := time.Now()
	endTime := deployItem.Status.TransitionTimes.InitTime.Time.Add(timeout.Duration)

	if currentTime.After(endTime) || currentTime.Equal(endTime) {
		msg := fmt.Sprintf("timeout at: %q", checkpoint)
		err := lserrors.NewError(op, lsv1alpha1.ProgressingTimeoutReason, msg, lsv1alpha1.ErrorTimeout)
		logger.Info(err.Error())
		return 0, err
	} else {
		remainingTime := endTime.Sub(currentTime)
		return remainingTime, nil
	}
}

func (t *standardTimeoutChecker) getTimeout(deployItem *lsv1alpha1.DeployItem) lsv1alpha1.Duration {
	timeout := deployItem.Spec.Timeout
	if timeout == nil {
		timeout = &lsv1alpha1.Duration{
			Duration: defaultTimeout,
		}
	}

	return *timeout
}
