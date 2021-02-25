// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/uuid"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/simplelogger"
)

type Options struct {
	// EnableCluster deploys a k8s cluster.
	// defaults to true
	EnableCluster bool
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

	// EnableRegistry also deploys a oci registry
	EnableRegistry bool
	// ExportRegistryCreds is the path to the file where the credentials for the registry should be written to.
	// The credentials are output as valid docker auth config.
	ExportRegistryCreds string
	// Password is the password that should be used for the registry baisc auth.
	// Will be generated if not provided
	Password string

	// OutputAddressFormat is the format of the output address for the registry
	// Can be either hostname or ip
	OutputAddressFormat string

	kubeClient client.Client
	restConfig *rest.Config
}

const (
	AddressFormatHostname = "hostname"
	AddressFormatIP       = "ip"
)

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

	if o.OutputAddressFormat != AddressFormatHostname && o.OutputAddressFormat != AddressFormatIP {
		return fmt.Errorf("unknown output format %q", o.OutputAddressFormat)
	}

	return nil
}

func main() {
	opts := &Options{}
	flag.BoolVar(&opts.EnableCluster, "enable-cluster", true, "deploy a cluster")
	flag.StringVar(&opts.HostClusterKubeconfigPath, "kubeconfig", "", "path to the host kubeconfig")
	flag.StringVarP(&opts.Namespace, "namespace", "n", "default", "namespace where the cluster should be created")
	flag.StringVar(&opts.ExportKubeconfigPath, "export", "", "path where the target kubeconfig should be written to")
	flag.StringVar(&opts.ID, "id", "", "unique id for the run. Will be generated and written to the state path if not specified.")
	flag.StringVar(&opts.StateFile, "state", "", "path where the state file should be written to")
	flag.DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "timeout for the command")

	flag.BoolVar(&opts.EnableRegistry, "enable-registry", false, "deploy a docker registry")
	flag.StringVar(&opts.Password, "registry-password", "", "set the registry password")
	flag.StringVar(&opts.ExportRegistryCreds, "registry-auth", "", "path where the docker auth config is written to")
	flag.StringVar(&opts.OutputAddressFormat, "address-format", "hostname", "the format of the addresses in the generated output. Can be hostname or ip.")
	flag.Parse()

	if err := run(flag.Args(), opts); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func (o *Options) Complete() error {
	var err error
	o.restConfig, err = clientcmd.BuildConfigFromFlags("", o.HostClusterKubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig from %s: %w", o.HostClusterKubeconfigPath, err)
	}
	o.kubeClient, err = client.New(o.restConfig, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client from %s: %w", o.HostClusterKubeconfigPath, err)
	}
	return nil
}

func (o *Options) initCreate() error {
	// generate id if none is defined
	if len(o.ID) == 0 {
		uid, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("unable to generate uuid: %w", err)
		}
		o.ID = base64.StdEncoding.EncodeToString([]byte(uid.String()))
	}
	if len(o.Password) == 0 {
		o.Password = RandString(10)
	}
	return nil
}

func (o *Options) initDelete() error {
	if len(o.ID) == 0 {
		// statefile should be defined as it is already checked by the calling function
		data, err := ioutil.ReadFile(o.StateFile)
		if err != nil {
			return fmt.Errorf("unable to read state file %q: %w", o.StateFile, err)
		}
		state := State{}
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("unable to decode state from %q: %w", o.StateFile, err)
		}
		o.ID = state.ID
	}
	return nil
}

func run(args []string, opts *Options) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 'create' or 'delete' but got %d arguments", len(args))
	}

	opts.ApplyDefault()
	if err := opts.Validate(); err != nil {
		return err
	}
	if err := opts.Complete(); err != nil {
		return err
	}

	ctx := context.Background()
	defer ctx.Done()
	logger := simplelogger.NewLogger().WithTimestamp()

	switch args[0] {
	case "create":
		if err := opts.initCreate(); err != nil {
			return err
		}
		if err := createCluster(ctx, logger, opts); err != nil {
			return err
		}
		return createRegistry(ctx, logger, opts)
	case "delete":
		if err := opts.initDelete(); err != nil {
			return err
		}
		if err := deleteCluster(ctx, logger, opts); err != nil {
			return err
		}
		return deleteRegistry(ctx, logger, opts)
	default:
		return fmt.Errorf("expected exactly 'create' or 'delete' but got %q", args[0])
	}
}
