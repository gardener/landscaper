// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/hack/testcluster/pkg"
)

// CommonOptions describes common options are used across different test cluster commands.
type CommonOptions struct {
	// Is the path to the host cluster where the cluster should be deployed
	HostClusterKubeconfigPath string
	// Namespace is the namespace where the cluster pod should be deployed to.
	Namespace string
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

// AddFlags adds flags for the options to a flagset
func (o *CommonOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}
	fs.StringVar(&o.HostClusterKubeconfigPath, "kubeconfig", "", "path to the host kubeconfig")
	fs.StringVarP(&o.Namespace, "namespace", "n", "default", "namespace where the cluster should be created")
	fs.StringVar(&o.ID, "id", "", "unique id for the run. Will be generated and written to the state path if not specified.")
	fs.StringVar(&o.StateFile, "state", "", "path where the state file should be written to")
	fs.DurationVar(&o.Timeout, "timeout", 10*time.Minute, "timeout for the command")
}

func (o *CommonOptions) Complete() error {
	var err error
	o.restConfig, err = clientcmd.BuildConfigFromFlags("", o.HostClusterKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig from %s: %w", o.HostClusterKubeconfigPath, err)
	}
	o.kubeClient, err = client.New(o.restConfig, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client from %s: %w", o.HostClusterKubeconfigPath, err)
	}

	if len(o.ID) == 0 {
		// statefile should be defined as it is already checked by the calling function
		data, err := os.ReadFile(o.StateFile)
		if err != nil {
			return fmt.Errorf("unable to read state file %q: %w", o.StateFile, err)
		}
		state := pkg.State{}
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("unable to decode state from %q: %w", o.StateFile, err)
		}
		o.ID = state.ID
	}
	return nil
}

func (o *CommonOptions) Validate() error {
	if len(o.HostClusterKubeconfigPath) == 0 {
		return errors.New("--kubeconfig has to be defined")
	}

	if len(o.ID) == 0 && len(o.StateFile) == 0 {
		return errors.New("either a unique id or state file have to be defined")
	}
	return nil
}

func NewTestClusterCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "testcluster",
	}

	cmd.AddCommand(NewClusterCommand(ctx))
	cmd.AddCommand(NewRegistryCommand(ctx))
	cmd.AddCommand(NewShootClusterCommand(ctx))
	return cmd
}
