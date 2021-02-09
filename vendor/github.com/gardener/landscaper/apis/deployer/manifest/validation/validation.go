// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/deployer/manifest"
)

// ValidateProviderConfiguration validates a manifest provider configuration.
func ValidateProviderConfiguration(config *manifest.ProviderConfiguration) error {
	var allErr field.ErrorList
	allErr = append(allErr, ValidateManifestList(field.NewPath(""), config.Manifests)...)

	return allErr.ToAggregate()
}

// ValidateManifestList validates a list of manifests
func ValidateManifestList(fldPath *field.Path, list []manifest.Manifest) field.ErrorList {
	var allErr field.ErrorList
	for i, m := range list {
		allErr = append(allErr, ValidateManifest(fldPath.Index(i), m)...)
	}
	return allErr
}

// ValidateManifest validates a manifest.
func ValidateManifest(fldPath *field.Path, manifest manifest.Manifest) field.ErrorList {
	var allErr field.ErrorList
	if manifest.Manifest == nil || len(manifest.Manifest.Raw) == 0 {
		allErr = append(allErr, field.Required(fldPath.Child("manifest"), "manifest must be defined"))
	}
	return allErr
}
