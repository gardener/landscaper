// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v3alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
)

// Validate validates a parsed v2 component descriptor.
func (cd *ComponentDescriptor) Validate() error {
	if err := Validate(nil, cd); err != nil {
		return errors.Wrapf(err.ToAggregate(), "%s:%s", cd.Name, cd.Version)
	}
	return nil
}

func Validate(fldPath *field.Path, component *ComponentDescriptor) field.ErrorList {
	if component == nil {
		return nil
	}
	allErrs := field.ErrorList{}

	if len(component.APIVersion) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("apiVersion"), "must specify a version"))
	}
	if len(component.Kind) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "must specify kind "+Kind))
	}
	if component.Kind != Kind {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("kind"), component.Kind, "must be "+Kind))
	}

	metaPath := fldPath.Child("metadata")
	if err := validateProvider(metaPath.Child("provider"), component.Provider); err != nil {
		allErrs = append(allErrs, err)
	}

	allErrs = append(allErrs, ValidateObjectMeta(metaPath, component)...)

	specPath := fldPath.Child("spec")
	srcPath := specPath.Child("sources")
	allErrs = append(allErrs, ValidateSources(srcPath, component.Spec.Sources)...)

	refPath := specPath.Child("references")
	allErrs = append(allErrs, ValidateComponentReferences(refPath, component.Spec.References)...)

	resourcePath := specPath.Child("resources")
	allErrs = append(allErrs, ValidateResources(resourcePath, component.Spec.Resources, component.GetVersion())...)

	return allErrs
}

// ValidateObjectMeta Validate the metadata of an object.
func ValidateObjectMeta(fldPath *field.Path, om compdesc.ObjectMetaAccessor) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(om.GetName()) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must specify a name"))
	}
	if len(om.GetVersion()) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("version"), "must specify a version"))
	}
	if len(om.GetLabels()) != 0 {
		allErrs = append(allErrs, metav1.ValidateLabels(fldPath.Child("labels"), om.GetLabels())...)
	}
	return allErrs
}

// ValidateSources validates a list of sources.
// It makes sure that no duplicate sources are present.
func ValidateSources(fldPath *field.Path, sources Sources) field.ErrorList {
	allErrs := field.ErrorList{}
	sourceIDs := make(map[string]struct{})
	for i, src := range sources {
		srcPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateSource(srcPath, src, false)...)

		id := src.GetIdentity(sources)
		dig := string(id.Digest())
		if _, ok := sourceIDs[dig]; ok {
			allErrs = append(allErrs, field.Duplicate(srcPath, fmt.Sprintf("duplicate source %s", id)))
			continue
		}
		sourceIDs[dig] = struct{}{}
	}
	return allErrs
}

// ValidateSource validates the a component's source object.
func ValidateSource(fldPath *field.Path, src Source, access bool) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(src.GetName()) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must specify a name"))
	}
	if len(src.GetType()) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must specify a type"))
	}
	if src.Access == nil && access {
		allErrs = append(allErrs, field.Required(fldPath.Child("access"), "must specify a access"))
	}
	allErrs = append(allErrs, metav1.ValidateIdentity(fldPath.Child("extraIdentity"), src.ExtraIdentity)...)
	return allErrs
}

// ValidateResource validates a components resource.
func ValidateResource(fldPath *field.Path, res Resource, access bool) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateObjectMeta(fldPath, &res)...)

	if err := metav1.ValidateRelation(fldPath.Child("relation"), res.Relation); err != nil {
		allErrs = append(allErrs, err)
	}

	if !metav1.IsIdentity(res.Name) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), res.Name, metav1.IdentityKeyValidationErrMsg))
	}

	if len(res.GetType()) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must specify a type"))
	}

	if res.Access == nil && access {
		allErrs = append(allErrs, field.Required(fldPath.Child("access"), "must specify a access"))
	}
	allErrs = append(allErrs, metav1.ValidateIdentity(fldPath.Child("extraIdentity"), res.ExtraIdentity)...)
	return allErrs
}

func validateProvider(fldPath *field.Path, provider metav1.Provider) *field.Error {
	if len(provider.Name) == 0 {
		return field.Required(fldPath.Child("name"), "provider name must be set")
	}
	return nil
}

// ValidateComponentReference validates a component version reference.
func ValidateComponentReference(fldPath *field.Path, cr Reference) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(cr.ComponentName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("componentName"), "must specify a component name"))
	}
	allErrs = append(allErrs, ValidateObjectMeta(fldPath, &cr)...)
	return allErrs
}

// ValidateComponentReferences validates a list of component version references.
// It makes sure that no duplicate sources are present.
func ValidateComponentReferences(fldPath *field.Path, refs References) field.ErrorList {
	allErrs := field.ErrorList{}
	refIDs := make(map[string]struct{})
	for i, ref := range refs {
		refPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateComponentReference(refPath, ref)...)

		id := ref.GetIdentity(refs)
		dig := string(id.Digest())
		if _, ok := refIDs[dig]; ok {
			allErrs = append(allErrs, field.Duplicate(refPath, fmt.Sprintf("duplicate component reference %s", id)))
			continue
		}
		refIDs[dig] = struct{}{}
	}
	return allErrs
}

// ValidateResources validates a list of resources.
// It makes sure that no duplicate sources are present.
func ValidateResources(fldPath *field.Path, resources Resources, componentVersion string) field.ErrorList {
	allErrs := field.ErrorList{}
	resourceIDs := make(map[string]struct{})
	for i, res := range resources {
		localPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateResource(localPath, res, true)...)

		if err := ValidateSourceRefs(localPath.Child("sourceRef"), res.SourceRef); err != nil {
			allErrs = append(allErrs, err...)
		}

		id := res.GetIdentity(resources)
		dig := string(id.Digest())
		if _, ok := resourceIDs[dig]; ok {
			allErrs = append(allErrs, field.Duplicate(localPath, fmt.Sprintf("duplicate resource %s", id)))
			continue
		}
		resourceIDs[dig] = struct{}{}
	}
	return allErrs
}

func ValidateSourceRefs(fldPath *field.Path, srcs []SourceRef) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, src := range srcs {
		localPath := fldPath.Index(i)
		if err := metav1.ValidateLabels(localPath.Child("labels"), src.Labels); err != nil {
			allErrs = append(allErrs, err...)
		}
	}
	return allErrs
}
