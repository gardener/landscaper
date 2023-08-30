// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/gardener/landscaper/controller-utils/pkg/logging/zapconfig"
)

type Config struct {
	flagset *flag.FlagSet

	Development       bool
	Cli               bool
	DisableStacktrace bool
	DisableCaller     bool
	DisableTimestamp  bool
	Level             logLevelValue
	Format            logFormatValue
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	fs := flag.NewFlagSet("log", flag.ExitOnError)

	fs.BoolVar(&configFromFlags.Development, "dev", false, "enable development logging")
	fs.BoolVar(&configFromFlags.Cli, "cli", false, "use CLI formatting for logs (color, no timestamps)")
	f := fs.VarPF(&configFromFlags.Format, "format", "f", "logging format [text, json]")
	f.DefValue = "text if either dev or cli flag is set, json otherwise"
	f = fs.VarPF(&configFromFlags.Level, "verbosity", "v", "logging verbosity [error, info, debug]")
	f.DefValue = "info, or debug if dev flag is set"
	fs.BoolVar(&configFromFlags.DisableStacktrace, "disable-stacktrace", true, "disable the stacktrace of error logs")
	fs.BoolVar(&configFromFlags.DisableCaller, "disable-caller", true, "disable the caller of logs")
	fs.BoolVar(&configFromFlags.DisableTimestamp, "disable-timestamp", false, "disable timestamp output")

	configFromFlags.flagset = fs
	flagset.AddFlagSet(configFromFlags.flagset)
}

// SetLogLevel sets the logging verbosity according to the provided flag if the flag was provided
func (c *Config) SetLogLevel(zapCfg *zapconfig.ZapConfig) {
	if !c.Level.IsUnset() {
		zapCfg.Level = zap.NewAtomicLevelAt(toZapLevel(c.Level.Value()))
	}
}

// SetLogFormat sets the logging format according to the provided flag if the flag was provided
func (c *Config) SetLogFormat(zapCfg *zapconfig.ZapConfig) {
	if !c.Format.IsUnset() {
		zapCfg.Encoding = toZapFormat(c.Format.Value())
	}
}

// SetDisableStacktrace dis- or enables the stackstrace according to the provided flag if the flag was provided
func (c *Config) SetDisableStacktrace(zapCfg *zapconfig.ZapConfig) {
	if c.flagset != nil && c.flagset.Changed("disable-stacktrace") {
		zapCfg.DisableStacktrace = c.DisableStacktrace
	}
}

// SetDisableCaller dis- or enables the caller according to the provided flag if the flag was provided
func (c *Config) SetDisableCaller(zapCfg *zapconfig.ZapConfig) {
	if c.flagset != nil && c.flagset.Changed("disable-caller") {
		zapCfg.DisableCaller = c.DisableCaller
	}
}

// SetTimestamp dis- or enables the logging of timestamps according to the provided flag if the flag was provided
func (c *Config) SetTimestamp(zapCfg *zapconfig.ZapConfig) {
	if c.flagset != nil && c.flagset.Changed("disable-timestamp") {
		if c.DisableTimestamp {
			zapCfg.EncoderConfig.TimeKey = ""
		} else {
			zapCfg.EncoderConfig.TimeKey = "ts"
		}
	}
}

// logLevelValue implements the Value interface for LogLevel
type logLevelValue struct {
	internal LogLevel
}

func (l *logLevelValue) String() string {
	return l.internal.String()
}
func (l *logLevelValue) Set(raw string) error {
	lvl, err := ParseLogLevel(raw)
	if err != nil {
		return err
	}
	l.internal = lvl
	return nil
}
func (l *logLevelValue) Type() string {
	return "LogLevel"
}
func (l *logLevelValue) Value() LogLevel {
	return l.internal
}
func (l *logLevelValue) IsUnset() bool {
	return l.internal == unknown_level
}
func (c *Config) WithLogLevel(l LogLevel) *Config {
	c.Level = logLevelValue{
		internal: l,
	}
	return c
}

// logFormatValue implements the Value interface for LogFormat
type logFormatValue struct {
	internal LogFormat
}

func (l *logFormatValue) String() string {
	return l.internal.String()
}
func (l *logFormatValue) Set(raw string) error {
	f, err := ParseLogFormat(raw)
	if err != nil {
		return err
	}
	l.internal = f
	return nil
}
func (l *logFormatValue) Type() string {
	return "LogFormat"
}
func (l *logFormatValue) Value() LogFormat {
	return l.internal
}
func (l *logFormatValue) IsUnset() bool {
	return l.internal == unknown_format
}
func (c *Config) WithLogFormat(f LogFormat) *Config {
	c.Format = logFormatValue{
		internal: f,
	}
	return c
}
