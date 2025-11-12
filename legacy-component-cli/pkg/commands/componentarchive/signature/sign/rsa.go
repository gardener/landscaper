// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package sign

import (
	"context"
	"errors"
	"fmt"
	"os"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
)

type RSASignOptions struct {
	// PathToPrivateKey for RSA signing
	PathToPrivateKey string

	GenericSignOptions
}

// NewGetCommand shows definitions and their configuration.
func NewRSASignCommand(ctx context.Context) *cobra.Command {
	opts := &RSASignOptions{}
	cmd := &cobra.Command{
		Use:   "rsa BASE_URL COMPONENT_NAME VERSION",
		Short: fmt.Sprintf("fetch the component descriptor from an oci registry or local filesystem, sign it using %s, and re-upload", cdv2.RSAPKCS1v15),
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

func (o *RSASignOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	signer, err := cdv2Sign.CreateRSASignerFromKeyFile(o.PathToPrivateKey, cdv2.MediaTypePEM)
	if err != nil {
		return fmt.Errorf("unable to create rsa signer: %w", err)
	}
	return o.SignAndUploadWithSigner(ctx, log, fs, signer)
}

func (o *RSASignOptions) Complete(args []string) error {
	if err := o.GenericSignOptions.Complete(args); err != nil {
		return err
	}

	if o.PathToPrivateKey == "" {
		return errors.New("a path to a private key file must be provided")
	}

	return nil
}

func (o *RSASignOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericSignOptions.AddFlags(fs)
	fs.StringVar(&o.PathToPrivateKey, "private-key", "", "path to private key file used for signing")
}
