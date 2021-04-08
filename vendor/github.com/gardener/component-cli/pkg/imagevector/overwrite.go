// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ComponentResolverFunc describes a function that can resolve a component descriptor by its name and version
type ComponentResolverFunc func(ctx context.Context, repoCtx cdv2.RepositoryContext, name, version string) (*cdv2.ComponentDescriptor, error)

// GenerateImageOverwrite parses a component descriptor and returns the defined image vector
func GenerateImageOverwrite(cd *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) (*ImageVector, error) {
	imageVector := &ImageVector{}

	// parse all images from the component descriptors resources
	images, err := parseImagesFromResources(cd.Resources)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	images, err = parseImagesFromComponentReferences(cd, list)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	images, err = parseGenericImages(cd, list)
	if err != nil {
		return nil, err
	}
	imageVector.Images = append(imageVector.Images, images...)

	return imageVector, nil
}

// parseImagesFromResources parse all images from the component descriptors resources
func parseImagesFromResources(resources []cdv2.Resource) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)
	for _, res := range resources {
		if res.GetType() != cdv2.OCIImageType || res.Access.GetType() != cdv2.OCIRegistryType {
			continue
		}
		var name string
		if ok, err := getLabel(res.GetLabels(), NameLabel, &name); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to get name for %q: %w", res.GetName(), err)
			}
			continue
		}

		entry := ImageEntry{
			Name: string(name),
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
func parseImagesFromComponentReferences(ca *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)

	for _, ref := range ca.ComponentReferences {

		// only resolve the component reference if a images.yaml is defined
		imageVector := &ImageVector{}
		if ok, err := getLabel(ref.GetLabels(), ImagesLabel, imageVector); !ok || err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to parse images label from component reference %q: %w", ref.GetName(), err)
			}
			continue
		}

		refCD, err := list.GetComponent(ref.ComponentName, ref.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component descriptor %q: %w", ref.GetName(), err)
		}

		// find the matching resource by name and version
		for _, image := range imageVector.Images {
			foundResources, err := refCD.GetResourcesByName(image.Name)
			if err != nil {
				return nil, fmt.Errorf("unable to find images for %q in component refernce %q: %w", image.Name, ref.GetName(), err)
			}
			for _, res := range foundResources {
				if res.GetVersion() != *image.Tag {
					continue
				}
				if err := parseResourceAccess(&image, res); err != nil {
					return nil, fmt.Errorf("unable to find images for %q in component refernce %q: %w", image.Name, ref.GetName(), err)
				}
				images = append(images, image)
			}
		}

	}

	return images, nil
}

// parseGenericImages parses the generic images of the component descriptor and matches all oci resources of the other component descriptors
func parseGenericImages(ca *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) ([]ImageEntry, error) {
	images := make([]ImageEntry, 0)
	imageVector := &ImageVector{}
	if ok, err := getLabel(ca.GetLabels(), ImagesLabel, imageVector); !ok || err != nil {
		if err != nil {
			return nil, fmt.Errorf("unable to parse images label from component reference %q: %w", ca.GetName(), err)
		}
		return images, nil
	}

	for _, image := range imageVector.Images {
		if image.TargetVersion == nil {
			continue
		}
		constr, err := semver.NewConstraint(*image.TargetVersion)
		if err != nil {
			return nil, fmt.Errorf("unable to parse target version for %q: %w", image.Name, err)
		}

		for _, comp := range list.Components {
			resources, err := comp.GetResourcesByType(cdv2.OCIImageType)
			if err != nil {
				if errors.Is(err, cdv2.NotFound) {
					continue
				}
				return nil, fmt.Errorf("unable to get oci resources from %q: %w", comp.GetName(), err)
			}
			for _, res := range resources {
				var imageName string
				ok, err := getLabel(res.GetLabels(), NameLabel, &imageName)
				if err != nil {
					return nil, fmt.Errorf("unable to parse image name label from resource %q of component %q: %w", res.GetName(), ca.GetName(), err)
				}
				if !ok || imageName != image.Name {
					continue
				}
				semverVersion, err := semver.NewVersion(res.GetVersion())
				if err != nil {
					return nil, fmt.Errorf("unable to parse resource version from resource %q of component %q: %w", res.GetName(), ca.GetName(), err)
				}
				if !constr.Check(semverVersion) {
					continue
				}

				entry := ImageEntry{
					Name: image.Name,
				}
				if err := parseResourceAccess(&entry, res); err != nil {
					return nil, fmt.Errorf("unable to parse oci access from resource %q of component %q: %w", res.GetName(), ca.GetName(), err)
				}
				targetVersion := fmt.Sprintf("= %s", *entry.Tag)
				entry.TargetVersion = &targetVersion
				images = append(images, entry)
			}
		}

	}

	return images, nil
}
