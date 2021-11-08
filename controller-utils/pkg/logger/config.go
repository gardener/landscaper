// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

type Config struct {
	flagset *flag.FlagSet

	Development       bool
	Cli               bool
	Verbosity         int
	DisableStacktrace bool
	DisableCaller     bool
	DisableTimestamp  bool
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	fs := flag.NewFlagSet("log", flag.ExitOnError)

	fs.BoolVar(&configFromFlags.Development, "dev", false, "enable development logging which result in console encoding, enabled stacktrace and enabled caller")
	fs.BoolVar(&configFromFlags.Cli, "cli", false, "logger runs as cli logger. enables cli logging")
	fs.IntVarP(&configFromFlags.Verbosity, "verbosity", "v", 1, "number for the log level verbosity")
	fs.BoolVar(&configFromFlags.DisableStacktrace, "disable-stacktrace", true, "disable the stacktrace of error logs")
	fs.BoolVar(&configFromFlags.DisableCaller, "disable-caller", true, "disable the caller of logs")
	fs.BoolVar(&configFromFlags.DisableTimestamp, "disable-timestamp", true, "disable timestamp output")

	configFromFlags.flagset = fs
	flagset.AddFlagSet(configFromFlags.flagset)
}

// SetDisableStacktrace dis- or enables the stackstrace according to the provided flag if the flag was provided
func (c *Config) SetDisableStacktrace(zapCfg *zap.Config) {
	if c.flagset == nil || c.flagset.Changed("disable-stacktrace") {
		zapCfg.DisableStacktrace = c.DisableStacktrace
	}
}

// SetDisableCaller dis- or enables the caller according to the provided flag if the flag was provided
func (c *Config) SetDisableCaller(zapCfg *zap.Config) {
	if c.flagset == nil || c.flagset.Changed("disable-caller") {
		zapCfg.DisableCaller = c.DisableCaller
	}
}

// SetTimestamp dis- or enables the logging of timestamps according to the provided flag if the flag was provided
func (c *Config) SetTimestamp(zapCfg *zap.Config) {
	if c.flagset == nil || c.flagset.Changed("disable-timestamp") {
		if c.DisableTimestamp {
			zapCfg.EncoderConfig.TimeKey = ""
		} else {
			zapCfg.EncoderConfig.TimeKey = "ts"
		}
	}
}
