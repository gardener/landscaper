package interruption

import (
	"context"
)

type ignoreInterruptionChecker struct{}

func NewIgnoreInterruptionChecker() InterruptionChecker {
	return &ignoreInterruptionChecker{}
}

func (*ignoreInterruptionChecker) Check(ctx context.Context) error {
	return nil
}
