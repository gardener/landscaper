// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"

	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

type options struct {
	log logging.Logger
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	logging.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logging.New(nil)
	if err != nil {
		return err
	}
	o.log = log.WithName("setup")
	logging.SetLogger(log)
	ctrl.SetLogger(log.Logr())

	return nil
}
