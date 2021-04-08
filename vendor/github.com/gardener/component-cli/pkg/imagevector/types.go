// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
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
	NameLabel             = Label("name")
	RepositoryLabel       = Label("repository")
	SourceRepositoryLabel = Label("source-repository")
	TargetVersionLabel    = Label("target-version")
	RuntimeVersionLabel   = Label("runtime-version")
	ImagesLabel           = Label("images")

	TagExtraIdentity = ExtraIdentityKey("tag")
)

// ImageVector defines a image vector that defines oci images with specific requirements
type ImageVector struct {
	Images []ImageEntry `json:"images"`
	Labels cdv2.Labels  `json:"labels,omitempty"`
}

// ImageEntry defines one image entry of a image vector
type ImageEntry struct {
	// Name defines the name of the image entry
	Name string `json:"name"`
	// SourceRepository is the name of the repository where the image was build from
	SourceRepository string `json:"sourceRepository,omitempty"`
	// Repository defines the image repository
	Repository string `json:"repository"`
	// +optional
	Tag *string `json:"tag,omitempty"`
	// +optional
	RuntimeVersion *string `json:"runtimeVersion,omitempty"`
	// +optional
	TargetVersion *string `json:"targetVersion,omitempty"`
	// +optional
	Labels cdv2.Labels `json:"labels,omitempty"`
}
