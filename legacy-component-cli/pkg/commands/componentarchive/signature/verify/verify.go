// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package verify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/commands/constants"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/signatures"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewVerifyCommand creates a new command to verify signatures.
func NewVerifyCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "command to verify the signature of a component descriptor",
	}

	cmd.AddCommand(NewRSAVerifyCommand(ctx))
	cmd.AddCommand(NewX509CertificateVerifyCommand(ctx))
	return cmd
}

type GenericVerifyOptions struct {
	// BaseUrl is the oci registry where the component is stored.
	BaseUrl string
	// ComponentName is the unique name of the component in the registry.
	ComponentName string
	// Version is the component version in the oci registry.
	Version string

	// SignatureName selects the matching signature to verify
	SignatureName string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options
}

// Complete validates the arguments and flags from the command line
func (o *GenericVerifyOptions) Complete(args []string) error {
	o.BaseUrl = args[0]
	o.ComponentName = args[1]
	o.Version = args[2]

	cliHomeDir, err := constants.CliHomeDir()
	if err != nil {
		return err
	}

	o.OciOptions.CacheDir = filepath.Join(cliHomeDir, "components")
	if err := os.MkdirAll(o.OciOptions.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %w", o.OciOptions.CacheDir, err)
	}

	if len(o.BaseUrl) == 0 {
		return errors.New("a base url must be provided")
	}
	if len(o.ComponentName) == 0 {
		return errors.New("a component name must be provided")
	}
	if len(o.Version) == 0 {
		return errors.New("a component version must be provided")
	}
	if o.SignatureName == "" {
		return errors.New("a signature name must be provided")
	}
	return nil
}

func (o *GenericVerifyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SignatureName, "signature-name", "", "name of the signature to verify")
	o.OciOptions.AddFlags(fs)
}

func (o *GenericVerifyOptions) VerifyWithVerifier(ctx context.Context, log logr.Logger, fs vfs.FileSystem, verifier cdv2Sign.Verifier) error {
	repoCtx := cdv2.NewOCIRegistryRepository(o.BaseUrl, "")

	ociClient, _, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}

	cdresolver := cdoci.NewResolver(ociClient)
	cd, err := cdresolver.Resolve(ctx, repoCtx, o.ComponentName, o.Version)
	if err != nil {
		return fmt.Errorf("unable to to fetch component descriptor %s:%s: %w", o.ComponentName, o.Version, err)
	}

	// check componentReferences and resources
	if err := CheckCdDigests(cd, *repoCtx, ociClient, context.TODO()); err != nil {
		return fmt.Errorf("unable to check component descriptor digests: %w", err)
	}

	// check if digest is correctly signed and the hash matches the normalised cd
	if err = cdv2Sign.VerifySignedComponentDescriptor(cd, verifier, o.SignatureName); err != nil {
		return fmt.Errorf("unable to verify signature: %w", err)
	}

	log.Info(fmt.Sprintf("Signature %s is valid and calculated digest matches existing digest", o.SignatureName))
	return nil
}

func CheckCdDigests(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient ociclient.Client, ctx context.Context) error {
	for _, reference := range cd.ComponentReferences {
		ociRef, err := cdoci.OCIRef(repoContext, reference.Name, reference.Version)
		if err != nil {
			return fmt.Errorf("unable to build oci reference from component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, err := cdresolver.Resolve(ctx, &repoContext, reference.ComponentName, reference.Version)
		if err != nil {
			return fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}

		if reference.Digest == nil || reference.Digest.HashAlgorithm == "" || reference.Digest.NormalisationAlgorithm == "" || reference.Digest.Value == "" {
			return fmt.Errorf("missing digest in component reference %s:%s", reference.ComponentName, reference.Version)
		}

		hasherForCdReference, err := cdv2Sign.HasherForName(reference.Digest.HashAlgorithm)
		if err != nil {
			return fmt.Errorf("unable to create hasher for component reference %s:%s: %w", reference.Name, reference.Version, err)
		}

		digest, err := recursivelyCheckCdsDigests(childCd, repoContext, ociClient, ctx, hasherForCdReference)
		if err != nil {
			return fmt.Errorf("unable to check digests for component reference %s:%s: %w", reference.ComponentName, reference.Version, err)
		}

		if !reflect.DeepEqual(reference.Digest, digest) {
			return fmt.Errorf("calculated digest mismatches existing digest for component reference %s:%s", reference.ComponentName, reference.Version)
		}
	}

	for _, resource := range cd.Resources {
		if resource.Access == nil || resource.Access.Type == "None" {
			if resource.Digest != nil {
				return fmt.Errorf("found access == nil or access.type == None in resource %s:%s", resource.Name, resource.Version)
			}
			continue
		}

		if resource.Digest == nil || resource.Digest.HashAlgorithm == "" || resource.Digest.NormalisationAlgorithm == "" || resource.Digest.Value == "" {
			return fmt.Errorf("missing digest in resource %s:%s", resource.Name, resource.Version)
		}

		hasher, err := cdv2Sign.HasherForName(resource.Digest.HashAlgorithm)
		if err != nil {
			return fmt.Errorf("unable to create hasher for resource %s:%s: %w", resource.Name, resource.Version, err)
		}
		digester := signatures.NewDigester(ociClient, *hasher)

		digest, err := digester.DigestForResource(ctx, *cd, resource)
		if err != nil {
			return fmt.Errorf("unable to calculate digest for resource %s:%s: %w", resource.Name, resource.Version, err)
		}

		if !reflect.DeepEqual(resource.Digest, digest) {
			return fmt.Errorf("calculated digest mismatches existing digest for resource %s:%s", resource.Name, resource.Version)
		}
	}

	return nil
}

func recursivelyCheckCdsDigests(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient ociclient.Client, ctx context.Context, hasherForCd *cdv2Sign.Hasher) (*cdv2.DigestSpec, error) {
	for referenceIndex, reference := range cd.ComponentReferences {
		reference := reference

		ociRef, err := cdoci.OCIRef(repoContext, reference.Name, reference.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to build oci reference from component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, err := cdresolver.Resolve(ctx, &repoContext, reference.ComponentName, reference.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}

		hasher, err := cdv2Sign.HasherForName(cdv2Sign.SHA256)
		if err != nil {
			return nil, fmt.Errorf("unable to create hasher for component reference %s:%s: %w", reference.Name, reference.Version, err)
		}

		digest, err := recursivelyCheckCdsDigests(childCd, repoContext, ociClient, ctx, hasher)
		if err != nil {
			return nil, fmt.Errorf("unable to check digests for component reference %s:%s: %w", reference.ComponentName, reference.Version, err)
		}
		reference.Digest = digest
		cd.ComponentReferences[referenceIndex] = reference
	}

	for resourceIndex, resource := range cd.Resources {
		resource := resource
		log := logger.Log.WithValues("componentDescriptor", cd, "resource.name", resource.Name, "resource.version", resource.Version, "resource.extraIdentity", resource.ExtraIdentity)

		hasher, err := cdv2Sign.HasherForName(cdv2Sign.SHA256)
		if err != nil {
			return nil, fmt.Errorf("unable to create hasher for resource %s:%s: %w", resource.Name, resource.Version, err)
		}

		digester := signatures.NewDigester(ociClient, *hasher)

		digest, err := digester.DigestForResource(ctx, *cd, resource)
		if err != nil {
			return nil, fmt.Errorf("unable to calculate digest for resource %s:%s: %w", resource.Name, resource.Version, err)
		}

		// For better user information, log resource with mismatching digest.
		// Since we do not trust the digest data in this cd, it is only for information purpose.
		// The mismatch will be noted in the propagated cd reference digest in the root cd.
		if resource.Digest != nil && !reflect.DeepEqual(resource.Digest, digest) {
			log.Info(fmt.Sprintf("calculated digest %+v mismatches existing (untrusted) digest %+v", digest, resource.Digest))
		}

		resource.Digest = digest
		cd.Resources[resourceIndex] = resource
	}

	hashCd, err := cdv2Sign.HashForComponentDescriptor(*cd, *hasherForCd)
	if err != nil {
		return nil, fmt.Errorf("unable to hash component descriptor %s:%s: %w", cd.Name, cd.Version, err)
	}

	return hashCd, nil
}
