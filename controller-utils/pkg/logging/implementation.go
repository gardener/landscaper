// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gardener/landscaper/controller-utils/pkg/logging/zapconfig"
)

var (
	Log             Logger
	AllLog          Logger
	configFromFlags = Config{}
	logConsumer     LogConsumer
)

type LogConsumer func(Logger)

func SetLogConsumer(c LogConsumer) {
	logConsumer = c
}

func encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func applyCLIEncoding(ecfg zapcore.EncoderConfig) zapcore.EncoderConfig {
	ecfg.TimeKey = ""
	ecfg.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	return ecfg
}

func defaultConfig() zapconfig.ZapConfig {
	return zapconfig.ZapConfig{
		Level:             zap.NewAtomicLevelAt(toZapLevel(INFO)),
		Development:       false,
		Encoding:          toZapFormat(TEXT),
		DisableStacktrace: true,
		DisableCaller:     true,
		EncoderConfig:     encoderConfig(),
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
	}
}

func applyCLIConfig(cfg zapconfig.ZapConfig) zapconfig.ZapConfig {
	cfg.EncoderConfig = applyCLIEncoding(cfg.EncoderConfig)
	return cfg
}

func applyDevConfig(cfg zapconfig.ZapConfig) zapconfig.ZapConfig {
	cfg.DisableCaller = false
	cfg.DisableStacktrace = false
	cfg.Development = true
	cfg.Level = zap.NewAtomicLevelAt(toZapLevel(DEBUG))
	return cfg
}

func applyProductionConfig(cfg zapconfig.ZapConfig) zapconfig.ZapConfig {
	cfg.Encoding = toZapFormat(JSON)
	return cfg
}

func newRootLogger() (Logger, Logger, error) {
	config := &configFromFlags

	zapCfg := determineZapConfig(config)

	zapLog, zapAll, err := zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return Logger{}, Logger{}, err
	}
	zapLog.Level()
	return Wrap(PreventKeyConflicts(zapr.NewLogger(zapLog))), Wrap(PreventKeyConflicts(zapr.NewLogger(zapAll))), nil
}

// GetLogger returns a singleton logger.
// Will initialize a new logger, if it doesn't exist yet.
func GetLogger() (Logger, error) {
	if Log.IsInitialized() {
		return Log, nil
	}
	log, all, err := newRootLogger()
	if err != nil {
		return Logger{}, err
	}
	SetLogger(log)
	SetAllLogger(all)
	if logConsumer != nil {
		logConsumer(all)
	}
	return log, nil
}

func SetLogger(log Logger) {
	Log = log
}
func SetAllLogger(log Logger) {
	AllLog = log
}

func determineZapConfig(loggerConfig *Config) zapconfig.ZapConfig {
	zapConfig := defaultConfig()
	if loggerConfig.Cli || loggerConfig.Development {
		if loggerConfig.Cli {
			zapConfig = applyCLIConfig(zapConfig)
		}
		if loggerConfig.Development {
			zapConfig = applyDevConfig(zapConfig)
		}
	} else {
		zapConfig = applyProductionConfig(zapConfig)
	}

	loggerConfig.SetLogLevel(&zapConfig)
	loggerConfig.SetLogFormat(&zapConfig)
	loggerConfig.SetDisableCaller(&zapConfig)
	loggerConfig.SetDisableStacktrace(&zapConfig)
	loggerConfig.SetTimestamp(&zapConfig)

	return zapConfig
}

func levelToVerbosity(level LogLevel) int {
	var res int
	switch level {
	case DEBUG:
		res = int(zap.DebugLevel)
	case ERROR:
		res = int(zap.ErrorLevel)
	default:
		res = int(zap.InfoLevel)
	}
	return res * -1
}

// toZapLevel converts our LogLevel into a zap Level.
// Unknown LogLevels are silently treated as INFO.
func toZapLevel(l LogLevel) zapcore.Level {
	switch l {
	case DEBUG:
		return zap.DebugLevel
	case ERROR:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

// toZapFormat converts our LogFormat into a zap encoding.
// Unknown LogFormats are silently treated as TEXT.
func toZapFormat(f LogFormat) string {
	switch f {
	case JSON:
		return "json"
	default:
		return "console"
	}
}
