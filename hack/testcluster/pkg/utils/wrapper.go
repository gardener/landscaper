package utils

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// NewLoggerFromTestLogger returns a logging.Logger which wraps the test framework logger defined in this package.
// It is intended for tests which must provide a logging.Logger, for example when they start an agent.
func NewLoggerFromTestLogger(internalLogger Logger) logging.Logger {
	s := &sink{
		internalLogger: internalLogger,
	}
	return logging.Wrap(logr.New(s).V(int(logging.DEBUG)))
}

type sink struct {
	internalLogger Logger
	name           string
	keysAndValues  []interface{}
}

var _ logr.LogSink = &sink{}

func (s *sink) Init(info logr.RuntimeInfo) {
}

func (s *sink) Enabled(_ int) bool {
	return true
}

func (s *sink) Info(_ int, msg string, keysAndValues ...interface{}) {
	ts := metav1.Now().Format(time.RFC3339)
	kv := append(s.keysAndValues, keysAndValues)
	s.internalLogger.Logln(fmt.Sprintf("level: info, ts: %s, logger: %s, msg: %s, %v", ts, s.name, msg, kv))
}

func (s *sink) Error(err error, msg string, keysAndValues ...interface{}) {
	ts := metav1.Now().Format(time.RFC3339)
	kv := append(s.keysAndValues, keysAndValues)
	s.internalLogger.Logln(fmt.Sprintf("level: error, ts: %s, logger: %s, msg: %s, error: %v, %v", ts, s.name, msg, err, kv))
}

func (s *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &sink{
		internalLogger: s.internalLogger,
		name:           s.name,
		keysAndValues:  append(s.keysAndValues, keysAndValues),
	}
}

func (s *sink) WithName(name string) logr.LogSink {
	return &sink{
		internalLogger: s.internalLogger,
		name:           name,
		keysAndValues:  s.keysAndValues,
	}
}
