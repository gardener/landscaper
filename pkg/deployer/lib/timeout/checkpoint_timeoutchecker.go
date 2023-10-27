package timeout

import (
	"context"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// checkpointTimeoutChecker is a TimeoutChecker whose TimeoutExceeded method raises a timeout error if the
// "checkpoint" argument has a certain value. This TimeoutChecker can be used in tests to provoke a timeout at a
// specific point in code.
type checkpointTimeoutChecker struct {
	checkpoint string
}

func newCheckpointTimeoutChecker(checkpoint string) *checkpointTimeoutChecker {
	return &checkpointTimeoutChecker{
		checkpoint: checkpoint,
	}
}

func (t *checkpointTimeoutChecker) TimeoutExceeded(ctx context.Context, deployItem *lsv1alpha1.DeployItem, checkpoint string) (time.Duration, lserrors.LsError) {
	logger, _ := logging.FromContextOrNew(ctx, nil)
	op := "TestTimeoutChecker.TimeoutExceeded"

	if t.checkpoint == checkpoint {
		err := lserrors.NewError(op, lsv1alpha1.ProgressingTimeoutReason, checkpoint, lsv1alpha1.ErrorTimeout)
		logger.Info(err.Error())
		return 0, err
	} else {
		return defaultTimeout, nil
	}
}
