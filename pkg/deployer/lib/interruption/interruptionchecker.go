package interruption

import (
	"context"
	"fmt"
)

var ErrInterruption = fmt.Errorf("processing of the deployitem was interrupted")

// InterruptionChecker is the interface to check for interrupts during the processing of a deployitem.
type InterruptionChecker interface {
	Check(ctx context.Context) error
}
