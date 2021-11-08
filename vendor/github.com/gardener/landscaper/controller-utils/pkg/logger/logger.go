// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
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
	DisableStacktrace: false,
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

	level := int8(0 - config.Verbosity)
	zapCfg.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

	zapLog, err := zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
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
	} else {
		zapConfig = productionConfig
	}

	loggerConfig.SetDisableCaller(&zapConfig)
	loggerConfig.SetDisableStacktrace(&zapConfig)
	loggerConfig.SetTimestamp(&zapConfig)

	return zapConfig
}
