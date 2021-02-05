// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

type Options struct {
	// Is the path to the host cluster where the cluster should be deployed
	HostClusterKubeconfigPath string
	// Namespace is the namespace where the cluster pod should be deployed to.
	Namespace string
	// ExportKubeconfigPath is the path to the test cluster kubeconfig
	ExportKubeconfigPath string
	// ID is the unique id for the current run.
	// +optional
	ID string
	// StateFile is the path where the state should be written to.
	// +optional
	StateFile string
	// Timeout timeout for the command.
	Timeout time.Duration

	kubeClient client.Client
	restConfig *rest.Config
}

// ApplyDefault sets defaults for the options
func (o *Options) ApplyDefault() {
	if len(o.HostClusterKubeconfigPath) == 0 {
		o.HostClusterKubeconfigPath = os.Getenv("KUBECONFIG")
	}
}

func (o *Options) Validate() error {
	if len(o.HostClusterKubeconfigPath) == 0 {
		return errors.New("--kubeconfig has to be defined")
	}

	if len(o.ID) == 0 && len(o.StateFile) == 0 {
		return errors.New("either a unique id or state file have to be defined")
	}

	return nil
}

func main() {
	opts := &Options{}
	flag.StringVar(&opts.HostClusterKubeconfigPath, "kubeconfig", "", "path to the host kubeconfig")
	flag.StringVarP(&opts.Namespace, "namespace", "n", "default", "namespace where the cluster should be created")
	flag.StringVar(&opts.ExportKubeconfigPath, "export", "", "path where the target kubeconfig should be written to")
	flag.StringVar(&opts.ID, "id", "", "unique id for the run. Will be generated and written to the state path if not specified.")
	flag.StringVar(&opts.StateFile, "state", "", "path where the state file should be written to")
	flag.DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "timeout for the command")
	flag.Parse()

	if err := run(flag.Args(), opts); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(args []string, opts *Options) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 'create' or 'delete' but got %d arguments", len(args))
	}

	opts.ApplyDefault()
	if err := opts.Validate(); err != nil {
		return err
	}

	ctx := context.Background()
	defer ctx.Done()
	logger := simplelogger.NewLogger().WithTimestamp()

	var err error
	opts.restConfig, err = clientcmd.BuildConfigFromFlags("", opts.HostClusterKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig from %s: %w", opts.HostClusterKubeconfigPath, err)
	}
	opts.kubeClient, err = client.New(opts.restConfig, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client from %s: %w", opts.HostClusterKubeconfigPath, err)
	}

	switch args[0] {
	case "create":
		return createCluster(ctx, logger, opts)
	case "delete":
		return deleteCluster(ctx, logger, opts)
	default:
		return fmt.Errorf("expected exactly 'create' or 'delete' but got %q", args[0])
	}
}
