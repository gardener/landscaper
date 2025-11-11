// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/oci"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"

	ociopts "github.com/gardener/landscaper/legacy-component-cli/ociclient/options"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

// CopyOptions contains all options to copy a component descriptor.
type CopyOptions struct {
	ComponentName    string
	ComponentVersion string
	SourceRepository string
	TargetRepository string

	// Recursive specifies if all component references should also be copied.
	Recursive bool
	// Force forces an overwrite in the target registry if the component descriptor is already uploaded.
	Force bool
	// CopyByValue defines if all oci images and artifacts should be copied by value or reference.
	// LocalBlobs are still copied by value.
	CopyByValue bool
	// KeepSourceRepository specifies if the source repository should be kept during the copy.
	// This value is only relevant if the artifacts are copied by value.
	KeepSourceRepository bool
	// TargetArtifactRepository is the target repository for oci artifacts.
	// This value is only relevant if the artifacts are copied by value.
	// +optional
	TargetArtifactRepository string
	// SourceArtifactRepository is the source repository for relative oci artifacts.
	// This value is only relevant if the artifacts are copied by value and if relative oci artifacts are copied.
	// The repository is defaulted to the "SourceRepository".
	// +optional
	SourceArtifactRepository string
	// ConvertToRelativeOCIReferences configures the cli to write copied artifacts back with a relative reference
	ConvertToRelativeOCIReferences bool

	// ReplaceOCIRefs contains replace expressions for manipulating upload refs of resources with accessType == ociRegistry
	ReplaceOCIRefs []string

	// OciOptions contains all exposed options to configure the oci client.
	OciOptions ociopts.Options

	MaxRetries    uint64
	BackoffFactor time.Duration
}

// NewCopyCommand creates a new definition command to push definitions
func NewCopyCommand(ctx context.Context) *cobra.Command {
	opts := &CopyOptions{}
	cmd := &cobra.Command{
		Use:   "copy COMPONENT_NAME VERSION --from SOURCE_REPOSITORY --to TARGET_REPOSITORY",
		Args:  cobra.ExactArgs(2),
		Short: "copies a component descriptor from a context repository to another",
		Long: `
copies a component descriptor and its blobs from the source repository to the target repository.

By default the component descriptor and all its component references are recursively copied.
This behavior can be overwritten by specifying "--recursive=false"

`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				logger.Log.Error(err, "")
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *CopyOptions) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	ctx = logr.NewContext(ctx, log)
	ociClient, cache, err := o.OciOptions.Build(log, fs)
	if err != nil {
		return fmt.Errorf("unable to build oci client: %s", err.Error())
	}
	defer cache.Close()

	replaceOCIRefs := map[string]string{}
	for _, replace := range o.ReplaceOCIRefs {
		splittedReplace := strings.Split(replace, ":")
		if len(splittedReplace) != 2 {
			return fmt.Errorf("invalid replace expression %s: must have the format left:right", replace)
		}
		replaceOCIRefs[splittedReplace[0]] = splittedReplace[1]
	}

	c := Copier{
		SrcRepoCtx:                     cdv2.NewOCIRegistryRepository(o.SourceRepository, ""),
		TargetRepoCtx:                  cdv2.NewOCIRegistryRepository(o.TargetRepository, ""),
		CompResolver:                   cdoci.NewResolver(ociClient),
		OciClient:                      ociClient,
		Cache:                          cache,
		Recursive:                      o.Recursive,
		Force:                          o.Force,
		CopyByValue:                    o.CopyByValue,
		KeepSourceRepository:           o.KeepSourceRepository,
		SourceArtifactRepository:       o.SourceArtifactRepository,
		TargetArtifactRepository:       o.TargetArtifactRepository,
		ConvertToRelativeOCIReferences: o.ConvertToRelativeOCIReferences,
		ReplaceOCIRefs:                 replaceOCIRefs,
		MaxRetries:                     o.MaxRetries,
		BackoffFactor:                  o.BackoffFactor,
	}

	if err := c.Copy(ctx, o.ComponentName, o.ComponentVersion); err != nil {
		return err
	}

	fmt.Printf("Successfully copied component descriptor %s:%s from %s to %s\n", o.ComponentName, o.ComponentVersion, o.SourceRepository, o.TargetRepository)
	return nil
}

func (o *CopyOptions) Complete(args []string) error {
	o.ComponentName = args[0]
	o.ComponentVersion = args[1]

	var err error
	o.OciOptions.CacheDir, err = utils.CacheDir()
	if err != nil {
		return fmt.Errorf("unable to get oci cache directory: %w", err)
	}

	if err := o.Validate(); err != nil {
		return err
	}
	if len(o.TargetArtifactRepository) == 0 {
		o.TargetArtifactRepository = o.TargetRepository
	}
	if len(o.SourceArtifactRepository) == 0 {
		o.SourceArtifactRepository = o.SourceRepository
	}
	return nil
}

// Validate validates push options
func (o *CopyOptions) Validate() error {
	if len(o.SourceRepository) == 0 {
		return errors.New("a source repository has to be specified")
	}
	if len(o.TargetRepository) == 0 {
		return errors.New("a target repository has to be specified")
	}
	return nil
}

func (o *CopyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.SourceRepository, "from", "", "source repository base url.")
	fs.StringVar(&o.TargetRepository, "to", "", "target repository where the components are copied to.")
	fs.BoolVar(&o.Recursive, "recursive", true, "Recursively copy the component descriptor and its references.")
	fs.BoolVar(&o.Force, "force", false, "Forces the tool to overwrite already existing component descriptors.")
	fs.BoolVar(&o.CopyByValue, "copy-by-value", false, "[EXPERIMENTAL] copies all referenced oci images and artifacts by value and not by reference.")
	fs.BoolVar(&o.KeepSourceRepository, "keep-source-repository", false, "Keep the original source repository when copying resources.")
	fs.StringVar(&o.TargetArtifactRepository, "target-artifact-repository", "",
		"target repository where the artifacts are copied to. This is only relevant if artifacts are copied by value and it will be defaulted to the target component repository")
	fs.StringVar(&o.SourceArtifactRepository, "source-artifact-repository", "",
		"source repository where relative oci artifacts are copied from. This is only relevant if artifacts are copied by value and it will be defaulted to the source component repository")
	fs.BoolVar(&o.ConvertToRelativeOCIReferences, "relative-urls", false, "converts all copied oci artifacts to relative urls")
	fs.StringSliceVar(&o.ReplaceOCIRefs, "replace-oci-ref", []string{}, "list of replace expressions in the format left:right. For every resource with accessType == "+cdv2.OCIRegistryType+", all occurences of 'left' in the target ref are replaced with 'right' before the upload")
	fs.Uint64Var(&o.MaxRetries, "max-retries", 0, "maximum number of retries for copying a component descriptor")
	fs.DurationVar(&o.BackoffFactor, "backoff-factor", 1*time.Second, "a backoff factor to apply between retry attempts: backoff = backoff-factor * 2^retries. e.g. if backoff-factor is 1s, then the timeouts will be [1s, 2s, 4s, …]")
	o.OciOptions.AddFlags(fs)
}

// Copier copies a component descriptor from a target repo to another.
type Copier struct {
	SrcRepoCtx, TargetRepoCtx cdv2.Repository
	Cache                     cache.Cache
	OciClient                 ociclient.Client
	CompResolver              ctf.ComponentResolver

	// Recursive specifies if all component references should also be copied.
	Recursive bool
	// Force forces an overwrite in the target registry if the component descriptor is already uploaded.
	Force bool
	// CopyByValue defines if all oci images and artifacts should be copied by value or reference.
	// LocalBlobs are still copied by value.
	CopyByValue bool
	// KeepSourceRepository specifies if the source repository should be kept during the copy.
	// This value is only relevant if the artifacts are copied by value.
	KeepSourceRepository bool
	// SourceArtifactRepository is the source repository for oci artifacts.
	// This value is only relevant if the artifacts are copied by value.
	SourceArtifactRepository string
	// TargetArtifactRepository is the target repository for oci artifacts.
	// This value is only relevant if the artifacts are copied by value.
	TargetArtifactRepository string
	// ConvertToRelativeOCIReferences configures the cli to write copied artifacts back with a relative reference
	ConvertToRelativeOCIReferences bool
	// ReplaceOCIRefs contains replace expressions for manipulating upload refs of resources with accessType == ociRegistry
	ReplaceOCIRefs map[string]string

	MaxRetries    uint64
	BackoffFactor time.Duration
}

func (c *Copier) copy(ctx context.Context, name, version string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("component", name, "version", version)
	log.Info("copy component descriptor")
	cd, blobs, err := c.CompResolver.ResolveWithBlobResolver(ctx, c.SrcRepoCtx, name, version)
	if err != nil {
		return err
	}

	if c.Recursive {
		log.V(5).Info("copy referenced components")
		for _, ref := range cd.ComponentReferences {
			if err := c.Copy(ctx, ref.ComponentName, ref.Version); err != nil {
				return err
			}
		}
	}

	// check if the component descriptor already exists
	if !c.Force && !c.CopyByValue {
		if _, err := c.CompResolver.Resolve(ctx, c.TargetRepoCtx, name, version); err == nil {
			log.V(3).Info("Component already exists. Nothing to copy.")
			return nil
		}
	}

	if err := cdv2.InjectRepositoryContext(cd, c.TargetRepoCtx); err != nil {
		return fmt.Errorf("unble to inject target repository: %w", err)
	}

	var layers []ocispecv1.Descriptor
	blobToResource := map[string]*cdv2.Resource{}
	// todo: parallelize upload with
	// todo: track if something has been uploaded otherwise only upload the component descriptor if "c.Force == true"
	for i, res := range cd.Resources {
		switch res.Access.Type {
		case cdv2.LocalOCIBlobType:
			localBlob := &cdv2.LocalOCIBlobAccess{}
			if err := res.Access.DecodeInto(localBlob); err != nil {
				return fmt.Errorf("unable to decode resource %s: %w", res.Name, err)
			}
			blobInfo, err := blobs.Info(ctx, res)
			if err != nil {
				return fmt.Errorf("unable to get blob info for resource %s: %w", res.Name, err)
			}
			d, err := digest.Parse(blobInfo.Digest)
			if err != nil {
				return fmt.Errorf("unable to parse digest for resource %s: %w", res.Name, err)
			}
			layers = append(layers, ocispecv1.Descriptor{
				MediaType: blobInfo.MediaType,
				Digest:    d,
				Size:      blobInfo.Size,
				Annotations: map[string]string{
					"resource": res.Name,
				},
			})
			blobToResource[blobInfo.Digest] = res.DeepCopy()
		case cdv2.OCIRegistryType:
			if !c.CopyByValue {
				log.V(7).Info("skip oci artifact copy by value", "resource", res.Name)
				continue
			}
			ociRegistryAcc := &cdv2.OCIRegistryAccess{}
			if err := res.Access.DecodeInto(ociRegistryAcc); err != nil {
				return fmt.Errorf("unable to decode resource %s: %w", res.Name, err)
			}

			// mangle the target artifact name to keep the original image ref somehow readable.
			target, err := targetOCIArtifactRef(c.TargetArtifactRepository, ociRegistryAcc.ImageReference, c.KeepSourceRepository)
			if err != nil {
				return fmt.Errorf("unable to create target oci artifact reference for resource %s: %w", res.Name, err)
			}

			for old, new := range c.ReplaceOCIRefs {
				target = strings.ReplaceAll(target, old, new)
			}

			log.V(4).Info(fmt.Sprintf("copy oci artifact %s to %s", ociRegistryAcc.ImageReference, target))
			if err := ociclient.Copy(ctx, c.OciClient, ociRegistryAcc.ImageReference, target); err != nil {
				return fmt.Errorf("unable to copy oci artifact %s from %s to %s: %w", res.Name, ociRegistryAcc.ImageReference, target, err)
			}

			if c.ConvertToRelativeOCIReferences {
				uAcc, err := cdv2.NewUnstructured(cdv2.NewRelativeOciAccess(strings.TrimPrefix(strings.TrimPrefix(target, c.TargetArtifactRepository), "/")))
				if err != nil {
					return fmt.Errorf("unable to marshal updated oci artifact access %s: %w", res.Name, err)
				}
				cd.Resources[i].Access = &uAcc
			} else {
				ociRegistryAcc.ImageReference = target
				uAcc, err := cdv2.NewUnstructured(ociRegistryAcc)
				if err != nil {
					return fmt.Errorf("unable to marshal updated oci artifact access %s: %w", res.Name, err)
				}
				cd.Resources[i].Access = &uAcc
			}

		case cdv2.RelativeOciReferenceType:
			if !c.CopyByValue {
				log.V(7).Info("skip relative oci artifact copy by value", "resource", res.Name)
				continue
			}
			relOCIRegistryAcc := &cdv2.RelativeOciAccess{}
			if err := res.Access.DecodeInto(relOCIRegistryAcc); err != nil {
				return fmt.Errorf("unable to decode resource %s: %w", res.Name, err)
			}

			src := path.Join(c.SourceArtifactRepository, relOCIRegistryAcc.Reference)
			target, err := targetOCIArtifactRef(c.TargetArtifactRepository, src, c.KeepSourceRepository)
			if err != nil {
				return fmt.Errorf("unable to create target oci artifact reference for resource %s: %w", res.Name, err)
			}

			for old, new := range c.ReplaceOCIRefs {
				target = strings.ReplaceAll(target, old, new)
			}

			log.V(4).Info(fmt.Sprintf("copy oci artifact %s to %s", src, target))
			if err := ociclient.Copy(ctx, c.OciClient, src, target); err != nil {
				return fmt.Errorf("unable to copy oci artifact %s from %s to %s: %w", res.Name, src, target, err)
			}

			if !c.ConvertToRelativeOCIReferences {
				uAcc, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryAccess(target))
				if err != nil {
					return fmt.Errorf("unable to marshal updated oci artifact access %s: %w", res.Name, err)
				}
				cd.Resources[i].Access = &uAcc
			}
		default:
			continue
		}
	}

	manifest, err := cdoci.NewManifestBuilder(c.Cache, ctf.NewComponentArchive(cd, nil)).Build(ctx)
	if err != nil {
		return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
	}
	manifest.Layers = append(manifest.Layers, layers...)

	ref, err := components.OCIRef(c.TargetRepoCtx, name, version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		log := log.WithValues("digest", desc.Digest.String(), "mediaType", desc.MediaType)
		res, ok := blobToResource[desc.Digest.String()]
		if !ok {
			// default to cache
			log.V(5).Info("copying resource from cache")
			rc, err := c.Cache.Get(desc)
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					log.Error(err, "unable to close blob reader")
				}
			}()
			if _, err := io.Copy(writer, rc); err != nil {
				return err
			}
			return nil
		}

		log.V(5).Info("copying resource", "resource", res.Name)
		_, err := blobs.Resolve(ctx, *res, writer)
		return err
	})

	log.V(3).Info("Upload component.", "ref", ref)
	if err := c.OciClient.PushManifest(ctx, ref, manifest, ociclient.WithStore(store)); err != nil {
		return err
	}

	return nil
}

func (c *Copier) Copy(ctx context.Context, name, version string) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("component", name, "version", version)

	for retries := uint64(0); retries <= c.MaxRetries; retries++ {
		err := c.copy(ctx, name, version)
		if err == nil {
			break
		}

		if err != nil && retries == c.MaxRetries {
			return fmt.Errorf("copy finished with error, max retries exceeded: %w", err)
		}

		backoff := utils.ExponentialBackoff(c.BackoffFactor, retries)
		log.Error(err, fmt.Sprintf("copy finished with error, retrying after %s ...", backoff))

		time.Sleep(backoff)
	}

	return nil
}

func targetOCIArtifactRef(targetRepo, ref string, keepOrigHost bool) (string, error) {
	if !strings.Contains(targetRepo, "://") {
		// add dummy protocol to correctly parse the url
		targetRepo = "http://" + targetRepo
	}
	t, err := url.Parse(targetRepo)
	if err != nil {
		return "", err
	}
	parsedRef, err := oci.ParseRef(ref)
	if err != nil {
		return "", err
	}

	if !keepOrigHost {
		parsedRef.Host = t.Host
		parsedRef.Repository = path.Join(t.Path, parsedRef.Repository)
		return parsedRef.String(), nil
	}
	replacedRef := strings.NewReplacer(".", "_", ":", "_").Replace(parsedRef.Name())
	parsedRef.Repository = path.Join(t.Path, replacedRef)
	parsedRef.Host = t.Host
	return parsedRef.String(), nil
}
