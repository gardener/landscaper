// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	goflag "flag"

	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// options holds the landscaper service controller options
type options struct {
	Log                      logging.Logger
	landscaperKubeconfigPath string
	installCrd               bool
}

// NewOptions returns a new options instance
func NewOptions() *options {
	return &options{}
}

// AddFlags adds flags passed via command line
func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.landscaperKubeconfigPath, "landscaper-kubeconfig", "", "Specify the path to the landscaper kubeconfig cluster")
	fs.BoolVar(&o.installCrd, "install-crd", false, "If true install CRDs")
	logging.InitFlags(fs)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete initializes the options instance and validates flags
func (o *options) Complete(ctx context.Context) error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.Log = log
	ctrl.SetLogger(log.Logr())

	if err != nil {
		return err
	}

	err = o.validate()
	return err
}

func (o *options) validate() error {
	return nil
}
