// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// LabelPrefix is the prefix for all image vector related labels on component descriptor resources.
const LabelPrefix = "imagevector.gardener.cloud"

// Label creates a new label for a name and append the image vector prefix.
func Label(name string) string {
	return fmt.Sprintf("%s/%s", LabelPrefix, name)
}

// ExtraIdentityKeyPrefix is the prefix for all image vector related extra identities on component descriptor resources.
const ExtraIdentityKeyPrefix = "imagevector-gardener-cloud"

// ExtraIdentityKey creates a new identity key for a name and append the image vector prefix.
func ExtraIdentityKey(name string) string {
	return fmt.Sprintf("%s+%s", ExtraIdentityKeyPrefix, name)
}

var (
	// ReferencedResourceNotFoundError is an error that indicates that a image is referenced by a external component
	// but it cannot be found in the referenced component.
	ReferencedResourceNotFoundError = errors.New("ReferencedResourceNotFound")
)

var (
	NameLabel             = Label("name")
	RepositoryLabel       = Label("repository")
	SourceRepositoryLabel = Label("source-repository")
	TargetVersionLabel    = Label("target-version")
	RuntimeVersionLabel   = Label("runtime-version")
	ImagesLabel           = Label("images")

	TagExtraIdentity        = ExtraIdentityKey("tag")
	RepositoryExtraIdentity = ExtraIdentityKey("repository")
)

// GardenerCIOriginalRefLabel describes the lable of the gardener ci that is used to identify the original ref of a resource.
const GardenerCIOriginalRefLabel = "cloud.gardener.cnudie/migration/original_ref"

// ImageVector defines a image vector that defines oci images with specific requirements
type ImageVector struct {
	Images []ImageEntry `json:"images"  yaml:"images,omitempty"`
	Labels cdv2.Labels  `json:"labels,omitempty"  yaml:"labels,omitempty"`
}

// ImageEntry defines one image entry of a image vector
type ImageEntry struct {
	// Name defines the name of the image entry
	Name string `json:"name" yaml:"name"`
	// SourceRepository is the name of the repository where the image was build from
	SourceRepository string `json:"sourceRepository,omitempty" yaml:"sourceRepository,omitempty"`
	// Repository defines the image repository
	Repository string `json:"repository" yaml:"repository,omitempty"`
	// +optional
	Tag *string `json:"tag,omitempty" yaml:"tag,omitempty"`
	// +optional
	RuntimeVersion *string `json:"runtimeVersion,omitempty" yaml:"runtimeVersion,omitempty"`
	// +optional
	TargetVersion *string `json:"targetVersion,omitempty" yaml:"targetVersion,omitempty"`
	// Labels describes optional labels that can be used to describe the image or add additional information.
	// +optional
	Labels cdv2.Labels `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// ComponentReferenceImageVector defines a image vector that defines oci images with specific requirements.
type ComponentReferenceImageVector struct {
	Images []ComponentReferenceImageEntry `json:"images"  yaml:"images,omitempty"`
}

// ComponentReferenceImageEntry defines one image entry of a image vector in a component reference
type ComponentReferenceImageEntry struct {
	ImageEntry
	// ResourceID is the name of the resource that the image references in the component descriptor.
	// +optional
	ResourceID cdv2.Identity `json:"resourceId,omitempty"`
}
