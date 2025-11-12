// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signatures

import (
	"context"
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"github.com/opencontainers/go-digest"

	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/components"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/logger"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	cdv2Sign "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2/signatures"
	cdoci "github.com/gardener/landscaper/legacy-component-spec/bindings-go/oci"

	ociCache "github.com/gardener/landscaper/legacy-component-cli/ociclient/cache"
)

func RecursivelyAddDigestsToCd(cd *cdv2.ComponentDescriptor, repoContext cdv2.OCIRegistryRepository, ociClient ociclient.Client, blobResolvers map[string]ctf.BlobResolver, ctx context.Context, skipAccessTypes map[string]bool) ([]*cdv2.ComponentDescriptor, error) {
	cdsWithHashes := []*cdv2.ComponentDescriptor{}

	cdResolver := func(c context.Context, cd cdv2.ComponentDescriptor, cr cdv2.ComponentReference) (*cdv2.DigestSpec, error) {
		ociRef, err := cdoci.OCIRef(repoContext, cr.Name, cr.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid component reference: %w", err)
		}

		cdresolver := cdoci.NewResolver(ociClient)
		childCd, blobResolver, err := cdresolver.ResolveWithBlobResolver(ctx, &repoContext, cr.ComponentName, cr.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to to fetch component descriptor %s: %w", ociRef, err)
		}
		blobResolvers[fmt.Sprintf("%s:%s", childCd.Name, childCd.Version)] = blobResolver

		cds, err := RecursivelyAddDigestsToCd(childCd, repoContext, ociClient, blobResolvers, ctx, skipAccessTypes)
		if err != nil {
			return nil, fmt.Errorf("failed resolving referenced cd %s:%s: %w", cr.Name, cr.Version, err)
		}
		cdsWithHashes = append(cdsWithHashes, cds...)

		hasher, err := cdv2Sign.HasherForName(cdv2Sign.SHA256)
		if err != nil {
			return nil, fmt.Errorf("failed creating hasher: %w", err)
		}
		hashCd, err := cdv2Sign.HashForComponentDescriptor(*childCd, *hasher)
		if err != nil {
			return nil, fmt.Errorf("failed hashing referenced cd %s:%s: %w", cr.Name, cr.Version, err)
		}
		return hashCd, nil
	}

	hasher, err := cdv2Sign.HasherForName(cdv2Sign.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed creating hasher: %w", err)
	}

	// set the do not sign digest notation on skip-access-type resources
	for i, res := range cd.Resources {
		res := res
		if _, ok := skipAccessTypes[res.Access.Type]; ok {
			log := logger.Log.WithValues("componentDescriptor", cd, "resource.name", res.Name, "resource.version", res.Version, "resource.extraIdentity", res.ExtraIdentity)
			log.Info(fmt.Sprintf("adding %s digest to resource based on skip-access-type", cdv2.ExcludeFromSignature))

			res.Digest = cdv2.NewExcludeFromSignatureDigest()
			cd.Resources[i] = res
		}
	}

	digester := NewDigester(ociClient, *hasher)
	if err := cdv2Sign.AddDigestsToComponentDescriptor(context.TODO(), cd, cdResolver, digester.DigestForResource); err != nil {
		return nil, fmt.Errorf("failed adding digests to cd %s:%s: %w", cd.Name, cd.Version, err)
	}
	cdsWithHashes = append(cdsWithHashes, cd)
	return cdsWithHashes, nil
}

func UploadCDPreservingLocalOciBlobs(ctx context.Context, cd cdv2.ComponentDescriptor, targetRepository cdv2.OCIRegistryRepository, ociClient ociclient.ExtendedClient, cache ociCache.Cache, blobResolvers map[string]ctf.BlobResolver, force bool, log logr.Logger) error {
	// check if the component descriptor already exists and skip if not forced to overwrite
	if !force {
		cdresolver := cdoci.NewResolver(ociClient)
		if _, err := cdresolver.Resolve(ctx, &targetRepository, cd.Name, cd.Version); err == nil {
			log.V(3).Info(fmt.Sprintf("Component Descriptor %s %s already exists in %s. Skip uploading cd", cd.Name, cd.Version, targetRepository.BaseURL))
			return nil
		}
	}

	if err := cdv2.InjectRepositoryContext(&cd, &targetRepository); err != nil {
		return fmt.Errorf("unble to inject target repository: %w", err)
	}

	// add all localOciBlobs to the layers
	var layers []ocispecv1.Descriptor
	blobToResource := map[string]*cdv2.Resource{}

	//get the blob resolver used for downloading
	blobResolver, ok := blobResolvers[fmt.Sprintf("%s:%s", cd.Name, cd.Version)]
	if !ok {
		return fmt.Errorf("no blob resolver found for %s %s", cd.Name, cd.Version)
	}

	for _, res := range cd.Resources {
		if res.Access.Type == cdv2.LocalOCIBlobType {
			localBlob := &cdv2.LocalOCIBlobAccess{}
			if err := res.Access.DecodeInto(localBlob); err != nil {
				return fmt.Errorf("unable to decode resource %s: %w", res.Name, err)
			}
			blobInfo, err := blobResolver.Info(ctx, res)
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

		}
	}
	manifest, err := cdoci.NewManifestBuilder(cache, ctf.NewComponentArchive(&cd, nil)).Build(ctx)
	if err != nil {
		return fmt.Errorf("unable to build oci artifact for component acrchive: %w", err)
	}
	manifest.Layers = append(manifest.Layers, layers...)

	ref, err := components.OCIRef(&targetRepository, cd.Name, cd.Version)
	if err != nil {
		return fmt.Errorf("invalid component reference: %w", err)
	}

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		log := log.WithValues("digest", desc.Digest.String(), "mediaType", desc.MediaType)
		res, ok := blobToResource[desc.Digest.String()]
		if !ok {
			// default to cache
			log.V(5).Info("copying resource from cache")
			rc, err := cache.Get(desc)
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
		_, err := blobResolver.Resolve(ctx, *res, writer)
		return err
	})
	log.V(3).Info("Upload component.", "ref", ref)
	if err := ociClient.PushManifest(ctx, ref, manifest, ociclient.WithStore(store)); err != nil {
		return fmt.Errorf("failed pushing manifest: %w", err)
	}
	return nil

}
