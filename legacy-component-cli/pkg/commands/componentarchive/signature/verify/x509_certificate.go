// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package verify

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/signatures"
)

type X509CertificateVerifyOptions struct {
	rootCACertPath          string
	intermediateCACertsPath string
	certPath                string

	GenericVerifyOptions
}

func NewX509CertificateVerifyCommand(ctx context.Context) *cobra.Command {
	opts := &X509CertificateVerifyOptions{}
	cmd := &cobra.Command{
		Use:   "x509 BASE_URL COMPONENT_NAME VERSION",
		Args:  cobra.ExactArgs(3),
		Short: fmt.Sprintf("fetch the component descriptor from an oci registry and verify its integrity based on a x509 certificate chain and a %s signature", cdv2.RSAPKCS1v15),
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

func (o *X509CertificateVerifyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	cert, err := signatures.CreateAndVerifyX509CertificateFromFiles(o.certPath, o.intermediateCACertsPath, o.rootCACertPath)
	if err != nil {
		return fmt.Errorf("unable to create certificate from files: %w", err)
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not of type *rsa.PublicKey: %T", cert.PublicKey)
	}

	verifier, err := cdv2Sign.CreateRSAVerifier(publicKey)
	if err != nil {
		return fmt.Errorf("unable to create rsa verifier: %w", err)
	}

	if err := o.VerifyWithVerifier(ctx, log, fs, verifier); err != nil {
		return fmt.Errorf("unable to verify component descriptor: %w", err)
	}
	return nil
}

func (o *X509CertificateVerifyOptions) Complete(args []string) error {
	if err := o.GenericVerifyOptions.Complete(args); err != nil {
		return err
	}

	if o.certPath == "" {
		return errors.New("a path to a certificate file must be provided")
	}

	return nil
}

func (o *X509CertificateVerifyOptions) AddFlags(fs *pflag.FlagSet) {
	o.GenericVerifyOptions.AddFlags(fs)
	fs.StringVar(&o.certPath, "cert", "", "path to a file containing the certificate file in PEM format")
	fs.StringVar(&o.intermediateCACertsPath, "intermediate-ca-certs", "", "[OPTIONAL] path to a file containing the concatenation of any intermediate ca certificates in PEM format")
	fs.StringVar(&o.rootCACertPath, "root-ca-cert", "", "[OPTIONAL] path to a file containing the root ca certificate in PEM format. if empty, the system root ca certificate pool is used")
}
