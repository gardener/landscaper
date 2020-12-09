// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/pkg/errors"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

func GetBlueprintDefinitionFromInstallationTemplate(inst *lsv1alpha1.Installation, subInstTmpl *lsv1alpha1.InstallationTemplate, cd *cdv2.ComponentDescriptor, cdResolveFunc cdutils.ResolveComponentReferenceFunc) (*lsv1alpha1.BlueprintDefinition, error) {
	subBlueprint := &lsv1alpha1.BlueprintDefinition{}
	// convert InstallationTemplateBlueprintDefinition to installation blueprint definition
	if len(subInstTmpl.Blueprint.Filesystem) != 0 {
		subBlueprint.Inline = &lsv1alpha1.InlineBlueprint{
			Filesystem: subInstTmpl.Blueprint.Filesystem,
		}
		if inst.Spec.Blueprint.Reference != nil {
			// uses the parent component descriptor
			subBlueprint.Inline.ComponentDescriptorReference = &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: inst.Spec.Blueprint.Reference.RepositoryContext,
				ComponentName:     inst.Spec.Blueprint.Reference.ComponentName,
				Version:           inst.Spec.Blueprint.Reference.Version,
			}
		}
	}
	if len(subInstTmpl.Blueprint.Ref) != 0 {
		uri, err := cdutils.ParseURI(subInstTmpl.Blueprint.Ref)
		if err != nil {
			return nil, err
		}
		if cd == nil {
			return nil, errors.New("no component descriptor defined to resolve the blueprint ref")
		}

		// resolve component descriptor list
		_, res, err := uri.Get(cd, cdResolveFunc)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve blueprint ref in component descriptor %s: %w", cd.Name, err)
		}
		// the result of the uri has to be an resource
		resource, ok := res.(cdv2.Resource)
		if !ok {
			return nil, fmt.Errorf("expected a resource from the component descriptor %s", cd.Name)
		}

		cd, err := uri.GetComponent(cd, cdResolveFunc)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component of blueprint ref in component descriptor %s: %w", cd.Name, err)
		}

		latestRepoCtx := cd.GetEffectiveRepositoryContext()
		subBlueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			VersionedResourceReference: lsv1alpha1.VersionedResourceReference{
				ResourceReference: lsv1alpha1.ResourceReference{
					ComponentName: cd.Name,
					ResourceName:  resource.Name,
				},
				Version: cd.Version,
			},
			RepositoryContext: &latestRepoCtx,
		}
	}

	return subBlueprint, nil
}

// getDefinitionReference returns the definition reference by name
func getDefinitionReference(blueprint *blueprints.Blueprint, name string) (*lsv1alpha1.InstallationTemplate, bool) {
	for _, ref := range blueprint.Subinstallations {
		if ref.Name == name {
			return ref, true
		}
	}
	return nil, false
}
