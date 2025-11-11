// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package sign

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/signatures"
)

type SigningServerSignOptions struct {
	ServerURL       string
	ClientCertPath  string
	PrivateKeyPath  string
	RootCACertsPath string

	GenericSignOptions
}

func NewSigningServerSignCommand(ctx context.Context) *cobra.Command {
	opts := &SigningServerSignOptions{}
	cmd := &cobra.Command{
		Use:   "signing-server BASE_URL COMPONENT_NAME VERSION",
		Short: "fetch the component descriptor from an oci registry or local filesystem, sign it with a signature provided from a signing server, and re-upload",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *SigningServerSignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	signer, err := signatures.NewSigningServerSigner(o.ServerURL, o.ClientCertPath, o.PrivateKeyPath, o.RootCACertsPath)
	if err != nil {
		return fmt.Errorf("unable to create signing server signer: %w", err)
	}
	return o.SignAndUploadWithSigner(ctx, log, fs, signer)
}

func (o *SigningServerSignOptions) Complete(args []string) error {
	if err := o.GenericSignOptions.Complete(args); err != nil {
		return err
	}

	if o.ServerURL == "" {
		return errors.New("a server url must be provided")
	}

	return nil
}

func (o *SigningServerSignOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericSignOptions.AddFlags(fs)
	fs.StringVar(&o.ServerURL, "server-url", "", "url where the signing server is running, e.g. https://localhost:8080")
	fs.StringVar(&o.ClientCertPath, "client-cert", "", "[OPTIONAL] path to a file containing the client certificate in PEM format for authenticating to the server")
	fs.StringVar(&o.PrivateKeyPath, "private-key", "", "[OPTIONAL] path to a file containing the private key for the provided client certificate in PEM format")
	fs.StringVar(&o.RootCACertsPath, "root-ca-certs", "", "[OPTIONAL] path to a file containing additional root ca certificates in PEM format. if empty, the system root ca certificate pool is used")
}
