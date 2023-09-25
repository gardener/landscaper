package timeout

import (
	"context"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

type TimeoutChecker interface {
	TimeoutExceeded(ctx context.Context, deployItem *lsv1alpha1.DeployItem, checkpoint string) (time.Duration, lserrors.LsError)
}

// timeoutCheckerInstance is the instance of the TimeoutChecker used by the deployers to detect progressing timeouts.
// Productively, we use the StandardTimeoutChecker. Some tests might replace it by another implementation.
var timeoutCheckerInstance TimeoutChecker = newStandardTimeoutChecker()

func ActivateStandardTimeoutChecker() {
	setTimeoutChecker(newStandardTimeoutChecker())
}

func ActivateIgnoreTimeoutChecker() {
	setTimeoutChecker(newIgnoreTimeoutChecker())
}

func ActivateCheckpointTimeoutChecker(checkpoint string) {
	setTimeoutChecker(newCheckpointTimeoutChecker(checkpoint))
}

func setTimeoutChecker(instance TimeoutChecker) {
	timeoutCheckerInstance = instance
}

// TimeoutExceeded checks whether the progressing timeout of a DeployItem has been exceeded.
func TimeoutExceeded(ctx context.Context, deployItem *lsv1alpha1.DeployItem, checkpoint string) (time.Duration, lserrors.LsError) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	op := "TimeoutExceeded"

	if timeoutCheckerInstance == nil {
		err := lserrors.NewError(op, "get timeout checker", "no timeout checker defined")
		logger.Error(err, err.Error())
		return 0, err
	}

	return timeoutCheckerInstance.TimeoutExceeded(ctx, deployItem, checkpoint)
}
