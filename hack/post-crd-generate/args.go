// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import "github.com/spf13/pflag"

var (
	dryRun = false
	debug  = false
)

func init() {
	pflag.BoolVar(&dryRun, "dry-run", false, "do not write changes to files, automatically enabled debug mode")
	pflag.BoolVar(&debug, "debug", false, "write crds to stdout")
}
