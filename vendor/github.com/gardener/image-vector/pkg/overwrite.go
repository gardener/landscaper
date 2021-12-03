// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// OCIResolver resolves oci references
type OCIResolver interface {
	// Resolve attempts to resolve the reference into a name and descriptor.
	//
	// The argument `ref` should be a scheme-less URI representing the remote.
	// Structurally, it has a host and path. The "host" can be used to directly
	// reference a specific host or be matched against a specific handler.
	//
	// The returned name should be used to identify the referenced entity.
	// Depending on the remote namespace, this may be immutable or mutable.
	// While the name may differ from ref, it should itself be a valid ref.
	//
	// If the resolution fails, an error will be returned.
	Resolve(ctx context.Context, ref string) (name string, desc ocispecv1.Descriptor, err error)
}

// GenerateImageOverwriteOptions are options to configure the image vector overwrite generation.
type GenerateImageOverwriteOptions struct {
	// Components defines a list of component descriptors that
	// should be used as source for the generic image dependencies.
	// +optional
	Components *cdv2.ComponentDescriptorList
	// ReplaceWithDigests configures the overwrite to automatically resolve tags to use digests.
	// If this is set to true the oci client is required
	ReplaceWithDigests bool
	// OciClient is a oci client to resolve references.
	OciClient OCIResolver
}

// Validate validates the GenerateImageOverwriteOptions.
func (o GenerateImageOverwriteOptions) Validate() error {
	if o.ReplaceWithDigests && o.OciClient == nil {
		return errors.New("a ociclient is required when tag should be replaced with digests")
	}
	return nil
}

// GenerateImageOverwrite parses a component descriptor and returns the defined image vector.
//
// Images can be defined in a component descriptor in 3 different ways:
// 1. as 'ociImage' resource: The image is defined a default resource of type 'ociImage' with a access of type 'ociRegistry'.
//    It is expected that the resource contains the following labels to be identified as image vector image.
//    The resulting image overwrite will contain the repository and the tag/digest from the access method.
// <pre>
// resources:
// - name: pause-container
//   version: "3.1"
//   type: ociImage
//   relation: external
//   extraIdentity:
//     "imagevector-gardener-cloud+tag": "3.1"
//   labels:
//   - name: imagevector.gardener.cloud/name
//     value: pause-container
//   - name: imagevector.gardener.cloud/repository
//     value: gcr.io/google_containers/pause-amd64
//   - name: imagevector.gardener.cloud/source-repository
//     value: github.com/kubernetes/kubernetes/blob/master/build/pause/Dockerfile
//   - name: imagevector.gardener.cloud/target-version
//     value: "< 1.16"
//   access:
//     type: ociRegistry
//     imageReference: gcr.io/google_containers/pause-amd64:3.1
// </pre>
//
// 2. as component reference: The images are defined in a label "imagevector.gardener.cloud/images".
//    The resulting image overwrite will contain all images defined in the images label.
//    Their repository and tag/digest will be matched from the resources defined in the actual component's resources.
//
//   Note: The images from the label are matched to the resources using their name and version. The original image reference do not exit anymore.
//
// <pre>
// componentReferences:
// - name: cluster-autoscaler-abc
//   componentName: github.com/gardener/autoscaler
//   version: v0.10.1
//   labels:
//   - name: imagevector.gardener.cloud/images
//     value:
//       images:
//       - name: cluster-autoscaler
//         repository: eu.gcr.io/gardener-project/gardener/autoscaler/cluster-autoscaler
//         tag: "v0.10.1"
// </pre>
//
// 3. as generic images from the component descriptor labels.
//   Generic images are images that do not directly result in a resource.
//   They will be matched with another component descriptor that actually defines the images.
//   The other component descriptor MUST have the "imagevector.gardener.cloud/name" label in order to be matched.
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// component:
//   labels:
//   - name: imagevector.gardener.cloud/images
//     value:
//       images:
//       - name: hyperkube
//         repository: k8s.gcr.io/hyperkube
//         targetVersion: "< 1.19"
// </pre>
//
// <pre>
// meta:
//   schemaVersion: 'v2'
// component:
//   resources:
//   - name: hyperkube
//     version: "v1.19.4"
//     type: ociImage
//     extraIdentity:
//       "imagevector-gardener-cloud+tag": "v1.19.4"
//     labels:
//     - name: imagevector.gardener.cloud/name
//       value: hyperkube
//     - name: imagevector.gardener.cloud/repository
//       value: k8s.gcr.io/hyperkube
//     access:
//	   type: ociRegistry
//	   imageReference: my-registry/hyperkube:v1.19.4
// </pre>
func GenerateImageOverwrite(ctx context.Context,
	compResolver ctf.ComponentResolver,
	cd *cdv2.ComponentDescriptor,
	opts GenerateImageOverwriteOptions) (*ImageVector, error) {
	log := logr.FromContextOrDiscard(ctx)

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	io := imageOverwrite{
		log:          log,
		compResolver: compResolver,
		opts:         opts,
	}

	imageVector := &ImageVector{}

	// parse all images from the component descriptors resources
	images, err := io.parseImagesFromResources(cd.Resources)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	images, err = io.parseImagesFromComponentReferences(ctx, cd)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	images, err = io.parseGenericImages(ctx, cd, opts.Components)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	if opts.ReplaceWithDigests {
		if err := resolveDigests(ctx, opts.OciClient, imageVector); err != nil {
			return nil, err
		}
	}

	return imageVector, nil
}

type imageOverwrite struct {
	log          logr.Logger
	compResolver ctf.ComponentResolver
	opts         GenerateImageOverwriteOptions
}

// parseImagesFromResources parse all images from the component descriptors resources
func (io *imageOverwrite) parseImagesFromResources(resources []cdv2.Resource) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)
	for _, res := range resources {
		log := io.log.WithValues("resource", res.Name)
		if res.GetType() != cdv2.OCIImageType || res.Access.GetType() != cdv2.OCIRegistryType {
			log.V(9).Info("ignoring non oci resource")
			continue
		}
		var name string
		if ok, err := getLabel(res.GetLabels(), NameLabel, &name); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get name for %q: %w", res.GetName(), err)
			}
			log.V(9).Info("ignoring resource without image vector overwrite label")
			continue
		}

		log.V(5).Info("create image entry from resource")
		entry := ImageEntry{
			Name: name,
		}

		if err := parseResourceAccess(&entry, res); err != nil {
			return nil, err
		}

		// set additional information
		var targetVersion string
		if ok, err := getLabel(res.GetLabels(), TargetVersionLabel, &targetVersion); ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get target version for %q: %w", res.GetName(), err)
			}
			entry.TargetVersion = &targetVersion
		}
		var runtimeVersion string
		if ok, err := getLabel(res.GetLabels(), RuntimeVersionLabel, &runtimeVersion); ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get target version for %q: %w", res.GetName(), err)
			}
			entry.RuntimeVersion = &runtimeVersion
		}

		images = append(images, entry)
	}
	return images, nil
}

// parseImagesFromComponentReferences parse all images from the component descriptors references
func (io *imageOverwrite) parseImagesFromComponentReferences(ctx context.Context, ca *cdv2.ComponentDescriptor) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)

	for _, ref := range ca.ComponentReferences {
		log := io.log.WithValues("component", ref.ComponentName, "version", ref.Version)
		// only resolve the component reference if a images.yaml is defined
		imageVector := &ComponentReferenceImageVector{}
		if ok, err := getLabel(ref.GetLabels(), ImagesLabel, imageVector); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to parse images label from component reference %q: %w", ref.GetName(), err)
			}
			log.V(9).Info("ignore component ref without a image vector label")
			continue
		}

		log.V(5).Info("create image entries from component ref")
		refCD, err := io.compResolver.Resolve(ctx, ca.GetEffectiveRepositoryContext(), ref.ComponentName, ref.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component descriptor %q: %w", ref.GetName(), err)
		}

		// find the matching resource by name and version
		for _, image := range imageVector.Images {
			imgLog := log.WithValues("image", image.Name, "imageVersion", image.Tag)
			imgLog.V(5).Info("create image entry for image")
			foundResources, err := getImageFromCompDesc(logr.NewContext(ctx, imgLog), image, refCD)
			if err != nil {
				return nil, fmt.Errorf("unable to find images for %q in component reference %q: %w", image.Name, ref.GetName(), err)
			}
			imgLog.V(7).Info(fmt.Sprintf("found %d images", len(foundResources)))
			resourceFound := false
			for _, res := range foundResources {
				if res.GetVersion() != *image.Tag {
					imgLog.V(7).Info("found resource version do not match", "version", *image.Tag, "expected", res.GetVersion())
					continue
				}
				if err := parseResourceAccess(&image.ImageEntry, res); err != nil {
					return nil, fmt.Errorf("unable to find images for %q in component reference %q: %w", image.Name, ref.GetName(), err)
				}
				resourceFound = true
				images = append(images, image.ImageEntry)
			}
			if !resourceFound {
				return nil, fmt.Errorf("unable to find images for %q in component reference %q: %w", image.Name, ref.GetName(), err)
			}
		}

	}

	return images, nil
}

func getImageFromCompDesc(ctx context.Context, image ComponentReferenceImageEntry, refCompDesc *cdv2.ComponentDescriptor) ([]cdv2.Resource, error) {
	log := logr.FromContextOrDiscard(ctx)
	// if a resource name is defined use this
	if len(image.ResourceID) != 0 {
		log.V(7).Info("match image with resource id", "resourceId", image.ResourceID)
		foundResource, err := refCompDesc.GetResourceByIdentity(image.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("unable to find images for %q in component reference %q", image.Name, image.ResourceID)
		}
		return []cdv2.Resource{foundResource}, nil
	}

	// otherwise first try to get the resource via image name
	foundResources, err := refCompDesc.GetResourcesByName(image.Name)
	if err == nil {
		return foundResources, nil
	}
	log.V(7).Info(fmt.Sprintf("unable to find image in component descriptor %s:%s by image name %s", refCompDesc.Name, refCompDesc.Version, image.Name), "err", err)

	for _, res := range refCompDesc.Resources {
		if res.Type != cdv2.OCIImageType {
			continue
		}
		// the ref can only be matched if the oci image is defined by a ociRegistry access
		if res.Access.GetType() != cdv2.OCIRegistryType {
			continue
		}

		// if not found try to match it via labels
		originalRef := ""
		if ok, err := getLabel(res.Labels, GardenerCIOriginalRefLabel, &originalRef); ok {
			if err != nil {
				log.Error(err, "unable to decode into oci registry", "resource", res.Name)
				continue
			}
			repo, _, err := ParseImageRef(originalRef)
			if err != nil {
				log.Error(err, "unable to parse image reference", "resource", res.Name, "ref", originalRef)
				continue
			}
			if repo == image.Repository {
				log.V(7).Info("match image with original image ref", "originalImageRef", originalRef)
				return []cdv2.Resource{res}, nil
			}
		}

		// or if not found try to match the repository
		acc := &cdv2.OCIRegistryAccess{}
		if err := res.Access.DecodeInto(acc); err != nil {
			log.Error(err, "unable to decode into oci registry", "resource", res.Name)
			continue
		}
		repo, _, err := ParseImageRef(acc.ImageReference)
		if err != nil {
			log.Error(err, "unable to parse image reference", "resource", res.Name, "ref", acc.ImageReference)
			continue
		}
		if repo == image.Repository {
			log.V(7).Info("match image with image repository", "repository", image.Repository)
			return []cdv2.Resource{res}, nil
		}
	}
	return nil, ReferencedResourceNotFoundError
}

// parseGenericImages parses the generic images of the component descriptor and matches all oci resources of the other component descriptors
func (io *imageOverwrite) parseGenericImages(ctx context.Context, ca *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) ([]ImageEntry, error) {
	io.log.V(5).Info("parse generic images")
	images := make([]ImageEntry, 0)
	imageVector := &ImageVector{}
	if ok, err := getLabel(ca.GetLabels(), ImagesLabel, imageVector); !ok || err != nil {
		if err != nil {
			return nil, fmt.Errorf("unable to parse images label from component reference %q: %w", ca.GetName(), err)
		}
		io.log.V(7).Info("no image vector label for generic images defined")
		return images, nil
	}

	for _, image := range imageVector.Images {
		if image.TargetVersion == nil {
			// it is expected that generic images without a target version are already handled as part of the default component descriptor.
			io.log.V(7).Info("ignore image with no target version", "image", image.Name)
			continue
		}
		entries, err := io.findGenericImageResource(ctx, image, list.Components)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("no corresponding resource found for %s", image.Name)
		}
		images = append(images, entries...)
	}

	return images, nil
}

func (io *imageOverwrite) findGenericImageResource(ctx context.Context, image ImageEntry, components []cdv2.ComponentDescriptor) ([]ImageEntry, error) {
	log := io.log.WithValues("image", image.Name)

	constr, err := semver.NewConstraint(*image.TargetVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to parse target version for %q: %w", image.Name, err)
	}

	log.V(7).Info("search corresponding resource for generic image in component descriptors")
	images := make([]ImageEntry, 0)
	for _, comp := range components {
		cLog := log.WithValues("component_name", comp.GetName(), "component_version", comp.GetVersion())

		resources, err := comp.GetResourcesByType(cdv2.OCIImageType)
		if err != nil {
			if errors.Is(err, cdv2.NotFound) {
				cLog.V(7).Info("no oci images found")
				continue
			}
			return nil, fmt.Errorf("unable to get oci resources from %q: %w", comp.GetName(), err)
		}

		cLog.V(7).Info("try to match found resources", "resources", len(resources))
		for _, res := range resources {
			rLog := cLog.WithValues("resource", res.Name)
			matches, err := resourceMatchesGenericImage(logr.NewContext(ctx, rLog), image, res)
			if err != nil {
				return nil, fmt.Errorf("unable to match image and resource %q of component %q: %w", res.GetName(), comp.GetName(), err)
			}
			if !matches {
				rLog.V(9).Info("neither the resource name nor the label name match the given image")
				continue
			}

			rLog.V(7).Info("add found resource for image")
			semverVersion, err := semver.NewVersion(res.GetVersion())
			if err != nil {
				return nil, fmt.Errorf("unable to parse resource version from resource %q of component %q: %w", res.GetName(), comp.GetName(), err)
			}
			if !constr.Check(semverVersion) {
				rLog.V(9).Info("semver constraint does not match", "version", res.GetVersion(), "constraint", *image.TargetVersion)
				continue
			}

			entry := ImageEntry{
				Name: image.Name,
			}
			if err := parseResourceAccess(&entry, res); err != nil {
				return nil, fmt.Errorf("unable to parse oci access from resource %q of component %q: %w", res.GetName(), comp.GetName(), err)
			}
			targetVersion := semverVersion.String()
			entry.TargetVersion = &targetVersion
			images = append(images, entry)
		}
	}
	return images, nil
}

func resourceMatchesGenericImage(ctx context.Context, image ImageEntry, res cdv2.Resource) (bool, error) {
	log := logr.FromContextOrDiscard(ctx)

	// this check should become the only check in the future
	// all subsequent checks are only intermediate fallbacks
	extraIdentityRepo, ok := res.GetIdentity()[RepositoryExtraIdentity]
	if ok && extraIdentityRepo == image.Repository {
		return true, nil
	}

	var imageName string
	_, err := getLabel(res.GetLabels(), NameLabel, &imageName)
	if err != nil {
		return false, fmt.Errorf("unable to parse image name label from resource %q: %w", res.GetName(), err)
	}
	if res.Name == image.Name || imageName == image.Name {
		return true, nil
	}

	// try to match the repository
	// if not found try to match it via labels
	originalRef := ""
	if ok, err := getLabel(res.Labels, GardenerCIOriginalRefLabel, &originalRef); ok {
		if err != nil {
			return false, fmt.Errorf("unable to get image label: %w", err)
		}
		repo, _, err := ParseImageRef(originalRef)
		if err != nil {
			return false, fmt.Errorf("unable to parse image reference: %w", err)
		}
		if repo == image.Repository {
			log.V(7).Info("match image with original image ref", "originalImageRef", originalRef)
			return true, nil
		}
	}

	// or if not found try to match the repository
	acc := &cdv2.OCIRegistryAccess{}
	if err := res.Access.DecodeInto(acc); err != nil {
		return false, fmt.Errorf("unable to decode into oci registry: %w", err)
	}
	repo, _, err := ParseImageRef(acc.ImageReference)
	if err != nil {
		return false, fmt.Errorf("unable to parse image reference %q: %w", acc.ImageReference, err)
	}
	if repo == image.Repository {
		return true, nil
	}

	return false, nil
}

// resolveDigests replaces all tags with their digest.
func resolveDigests(ctx context.Context, ociClient OCIResolver, iv *ImageVector) error {
	for i, img := range iv.Images {
		if TagIsDigest(*img.Tag) {
			continue
		}
		ref := img.Repository + ":" + *img.Tag
		_, desc, err := ociClient.Resolve(ctx, ref)
		if err != nil {
			return fmt.Errorf("unable to resolve digest for %q: %w", ref, err)
		}

		dig := desc.Digest.String()
		iv.Images[i].Tag = &dig
	}
	return nil
}
