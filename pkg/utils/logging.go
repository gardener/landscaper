package utils

import (
	"github.com/go-logr/logr"
	"golang.org/x/net/context"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// FromContext wraps the result of logr.FromContext into a logging.Logger. If no logger exists a new one is created.
// If a new logger has to be created, it logs with the provided keys and valuers
func FromContextOrNew(ctx context.Context, keysAndValuesForNewLogger ...interface{}) (logging.Logger, context.Context) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		newLogger, err := logging.GetLogger()
		if err != nil {
			panic(err)
		}

		newLogger = newLogger.WithValues("CreatedBy", "FromContextOrNew", keysAndValuesForNewLogger)
		ctx = logging.NewContext(ctx, newLogger)
		return newLogger, ctx
	} else {
		wrappedLogger := logging.Wrap(log)
		ctx = logging.NewContext(ctx, wrappedLogger)
		return wrappedLogger, ctx
	}
}
