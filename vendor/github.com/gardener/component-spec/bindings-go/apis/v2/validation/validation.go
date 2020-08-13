// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// Validate validates a parsed v2 component descriptor
func Validate(component *v2.ComponentDescriptor) error {
	if err := validate(nil, component); err != nil {
		return err.ToAggregate()
	}
	return nil
}

func validate(fldPath *field.Path, component *v2.ComponentDescriptor) field.ErrorList {
	if component == nil {
		return nil
	}
	allErrs := field.ErrorList{}

	if len(component.Metadata.Version) == 0 {
		metaPath := field.NewPath("meta").Child("schemaVersion")
		if fldPath != nil {
			metaPath = fldPath.Child("meta").Child("schemaVersion")
		}
		allErrs = append(allErrs, field.Required(metaPath, "must specify a version"))
	}

	compPath := field.NewPath("component")
	if fldPath != nil {
		compPath = fldPath.Child("component")
	}

	if err := validateProvider(compPath.Child("provider"), component.Provider); err != nil {
		allErrs = append(allErrs, err)
	}

	allErrs = append(allErrs, validateObjectMeta(compPath, component.ObjectMeta)...)

	srcPath := compPath.Child("sources")
	for i, src := range component.Sources {
		allErrs = append(allErrs, validateResource(srcPath.Index(i), src)...)
	}

	refPath := compPath.Child("componentReferences")
	for i, ref := range component.ComponentReferences {
		allErrs = append(allErrs, validateObjectMeta(refPath.Index(i), ref)...)
	}

	localPath := compPath.Child("localResources")
	for i, res := range component.LocalResources {
		allErrs = append(allErrs, validateResource(localPath.Index(i), res)...)
		if res.GetVersion() != component.GetVersion() {
			allErrs = append(allErrs, field.Invalid(localPath.Index(i).Child("version"), "invalid version",
				"version of local resources must match the component version"))
		}
	}

	extPath := compPath.Child("externalResources")
	for i, res := range component.ExternalResources {
		allErrs = append(allErrs, validateResource(extPath.Index(i), res)...)
	}

	return allErrs
}

func validateObjectMeta(fldPath *field.Path, om v2.ObjectMeta) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(om.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must specify a name"))
	}
	if len(om.Version) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("version"), "must specify a version"))
	}
	return allErrs
}

func validateResource(fldPath *field.Path, res v2.Resource) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateObjectMeta(fldPath, res.ObjectMeta)...)

	if len(res.Type) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must specify a type"))
	}

	return allErrs
}

func validateProvider(fldPath *field.Path, provider v2.ProviderType) *field.Error {
	if len(provider) == 0 {
		return field.Required(fldPath, "provider must be set and one of (internal, external)")
	}
	if provider != v2.InternalProvider && provider != v2.ExternalProvider {
		return field.Invalid(fldPath, "unknown provider type", "provider must be one of (internal, external)")
	}
	return nil
}
