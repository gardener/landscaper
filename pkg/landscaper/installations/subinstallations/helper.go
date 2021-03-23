// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/pkg/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// GetBlueprintDefinitionFromInstallationTemplate returns a Blueprint and a ComponentDescriptor from a subinstallation
func GetBlueprintDefinitionFromInstallationTemplate(inst *lsv1alpha1.Installation, subInstTmpl *lsv1alpha1.InstallationTemplate, cd *cdv2.ComponentDescriptor, cdResolveFunc cdutils.ResolveComponentReferenceFunc) (*lsv1alpha1.BlueprintDefinition, *lsv1alpha1.ComponentDescriptorDefinition, error) {
	subBlueprint := &lsv1alpha1.BlueprintDefinition{}

	//store reference to parent component descriptor
	var cdDef *lsv1alpha1.ComponentDescriptorDefinition = inst.Spec.ComponentDescriptor

	// convert InstallationTemplateBlueprintDefinition to installation blueprint definition
	if len(subInstTmpl.Blueprint.Filesystem.RawMessage) != 0 {
		subBlueprint.Inline = &lsv1alpha1.InlineBlueprint{
			Filesystem: subInstTmpl.Blueprint.Filesystem,
		}
	}

	if len(subInstTmpl.Blueprint.Ref) != 0 {
		uri, err := cdutils.ParseURI(subInstTmpl.Blueprint.Ref)
		if err != nil {
			return nil, nil, err
		}
		if cd == nil {
			return nil, nil, errors.New("no component descriptor defined to resolve the blueprint ref")
		}

		// resolve component descriptor list
		_, res, err := uri.Get(cd, cdResolveFunc)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve blueprint ref in component descriptor %s: %w", cd.Name, err)
		}
		// the result of the uri has to be an resource
		resource, ok := res.(cdv2.Resource)
		if !ok {
			return nil, nil, fmt.Errorf("expected a resource from the component descriptor %s", cd.Name)
		}

		cd, err := uri.GetComponent(cd, cdResolveFunc)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve component of blueprint ref in component descriptor %s: %w", cd.Name, err)
		}

		// remove parent component descriptor
		cdDef = &lsv1alpha1.ComponentDescriptorDefinition{}

		// check, if the component descriptor of the subinstallation has been defined as a nested inline CD in the parent installation
		if inst.Spec.ComponentDescriptor != nil && inst.Spec.ComponentDescriptor.Inline != nil {
			for _, ref := range inst.Spec.ComponentDescriptor.Inline.ComponentReferences {
				if ref.ComponentName == cd.GetName() && ref.Version == cd.GetVersion() {
					if label, exists := ref.Labels.Get(lsv1alpha1.InlineComponentDescriptorLabel); exists {
						// unmarshal again form parent installation to retain all levels of nested component descriptors
						var cdFromLabel cdv2.ComponentDescriptor
						if err := codec.Decode(label, &cdFromLabel); err != nil {
							return nil, nil, err
						}
						cdDef.Inline = &cdFromLabel
					}
				}
			}
		}

		if cdDef.Inline == nil {
			latestRepoCtx := cd.GetEffectiveRepositoryContext()
			cdDef = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: &latestRepoCtx,
					ComponentName:     cd.Name,
					Version:           cd.Version,
				},
			}
		}

		subBlueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resource.Name,
		}
	}

	return subBlueprint, cdDef, nil
}

// getSubinstallationTemplate returns the installation template by name
func getSubinstallationTemplate(blueprint *blueprints.Blueprint, name string) (*lsv1alpha1.InstallationTemplate, bool) {
	for _, ref := range blueprint.Subinstallations {
		if ref.Name == name {
			return ref, true
		}
	}
	return nil, false
}
