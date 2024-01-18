package utils

import (
	"strconv"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

const (
	keyDescription    = "description"
	keyDuration       = "duration"
	keyDurationMillis = "durationMillis"
)

type PerformanceMeasurement struct {
	log         *logging.Logger
	description string
	startTime   time.Time
}

func StartPerformanceMeasurement(log *logging.Logger, description string) *PerformanceMeasurement {
	log.Info("start performance measurement", keyDescription, description)
	return &PerformanceMeasurement{
		log:         log,
		description: description,
		startTime:   time.Now(),
	}
}

func (p *PerformanceMeasurement) Stop() {
	duration := time.Since(p.startTime)
	durationMillis := strconv.FormatInt(duration.Milliseconds(), 10)
	p.log.Info("stop performance measurement",
		keyDescription, p.description,
		keyDuration, duration.String(),
		keyDurationMillis, durationMillis)
}
