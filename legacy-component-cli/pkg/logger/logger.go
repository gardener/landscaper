// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log             logr.Logger
	configFromFlags = Config{}
)

var encoderConfig = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "msg",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

var cliEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "msg",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

var defaultConfig = zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       true,
	Encoding:          "console",
	DisableStacktrace: false,
	DisableCaller:     false,
	EncoderConfig:     encoderConfig,
	OutputPaths:       []string{"stderr"},
	ErrorOutputPaths:  []string{"stderr"},
}

var cliConfig = zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       false,
	Encoding:          "console",
	DisableStacktrace: true,
	DisableCaller:     true,
	EncoderConfig:     cliEncoderConfig,
	OutputPaths:       []string{"stderr"},
	ErrorOutputPaths:  []string{"stderr"},
}

var productionConfig = zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       false,
	DisableStacktrace: true,
	DisableCaller:     true,
	Encoding:          "json",
	EncoderConfig:     encoderConfig,
	OutputPaths:       []string{"stderr"},
	ErrorOutputPaths:  []string{"stderr"},
}

func New(config *Config) (logr.Logger, error) {
	if config == nil {
		config = &configFromFlags
	}
	zapCfg := determineZapConfig(config)

	zapLog, err := zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return logr.Logger{}, err
	}
	return zapr.NewLogger(zapLog), nil
}

func SetLogger(log logr.Logger) {
	Log = log
}

// NewCliLogger creates a new logger for cli usage.
// CLI usage means that by default:
// - the default dev config
// - encoding is console
// - timestamps are disabled (can be still activated by the cli flag)
// - level are color encoded
func NewCliLogger() (logr.Logger, error) {
	config := &configFromFlags
	config.Cli = true
	return New(config)
}

func determineZapConfig(loggerConfig *Config) zap.Config {
	var zapConfig zap.Config
	if loggerConfig.Development {
		zapConfig = defaultConfig
	} else if loggerConfig.Cli {
		zapConfig = cliConfig
		if loggerConfig.Development {
			zapConfig.Development = true
			loggerConfig.DisableCaller = false
		}
		// only enable the stacktrace for a verbosity > 4
		if loggerConfig.Verbosity > 4 {
			loggerConfig.DisableStacktrace = false
		}
	} else {
		zapConfig = productionConfig
	}

	loggerConfig.SetDisableCaller(&zapConfig)
	loggerConfig.SetDisableStacktrace(&zapConfig)
	loggerConfig.SetTimestamp(&zapConfig)

	if len(os.Getenv(LoggingVerbosityEnvVar)) != 0 {
		var err error
		loggerConfig.Verbosity, err = strconv.Atoi(os.Getenv(LoggingVerbosityEnvVar))
		if err != nil {
			panic(fmt.Sprintf("unable to convert %s %s to int", LoggingVerbosityEnvVar, os.Getenv(LoggingVerbosityEnvVar)))
		}
	}
	level := int8(0 - loggerConfig.Verbosity)
	zapConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

	return zapConfig
}
