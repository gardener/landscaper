// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"fmt"

	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// GetBlueprintDefinitionFromInstallationTemplate returns a Blueprint and a ComponentDescriptor from a subinstallation
func GetBlueprintDefinitionFromInstallationTemplate(
	inst *lsv1alpha1.Installation,
	subInstTmpl *lsv1alpha1.InstallationTemplate,
	componentVersion model.ComponentVersion,
	repositoryContext *types.UnstructuredTypedObject,
	overwriter componentoverwrites.Overwriter) (*lsv1alpha1.BlueprintDefinition, *lsv1alpha1.ComponentDescriptorDefinition, error) {
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
		if componentVersion == nil {
			return nil, nil, errors.New("no component descriptor defined to resolve the blueprint ref")
		}

		// resolve component descriptor list
		_, res, err := uri.Get(componentVersion, repositoryContext)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve blueprint ref in component descriptor %s: %w", componentVersion.GetName(), err)
		}
		// the result of the uri has to be a resource
		resource, ok := res.(model.Resource)
		if !ok {
			return nil, nil, fmt.Errorf("expected a resource from the component descriptor %s", componentVersion.GetName())
		}

		subInstCompVers, subInstCdRef, err := uri.GetComponent(componentVersion, repositoryContext, overwriter)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve component of blueprint ref in component descriptor %s: %w", componentVersion.GetName(), err)
		}

		// remove parent component descriptor
		cdDef = &lsv1alpha1.ComponentDescriptorDefinition{}

		// check, if the component descriptor of the subinstallation has been defined as a nested inline CD in the parent installation
		if inst.Spec.ComponentDescriptor != nil && inst.Spec.ComponentDescriptor.Inline != nil {
			for _, ref := range inst.Spec.ComponentDescriptor.Inline.ComponentReferences {
				if ref.ComponentName == subInstCompVers.GetName() && ref.Version == subInstCompVers.GetVersion() {
					if label, exists := ref.Labels.Get(lsv1alpha1.InlineComponentDescriptorLabel); exists {
						// unmarshal again form parent installation to retain all levels of nested component descriptors
						var cdFromLabel types.ComponentDescriptor
						if err := codec.Decode(label, &cdFromLabel); err != nil {
							return nil, nil, err
						}
						cdDef.Inline = &cdFromLabel
					}
				}
			}
		}

		if cdDef.Inline == nil {
			cdDef = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: subInstCdRef,
			}
		}

		subBlueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resource.GetName(),
		}
	}

	return subBlueprint, cdDef, nil
}

// ValidateSubinstallations validates the installation templates in context of the current blueprint.
func (o *Operation) ValidateSubinstallations(installationTmpl []*lsv1alpha1.InstallationTemplate) error {
	coreInstTmpls, err := convertToCoreInstallationTemplates(installationTmpl)
	if err != nil {
		return err
	}

	coreImports, err := o.buildCoreImports(o.Inst.GetBlueprint().Info.Imports)
	if err != nil {
		return err
	}

	if allErrs := validation.ValidateInstallationTemplates(field.NewPath("subinstallations"), coreImports, coreInstTmpls); len(allErrs) != 0 {
		return o.NewError(allErrs.ToAggregate(), "ValidateSubInstallations", allErrs.ToAggregate().Error())
	}
	return nil
}

// convertToCoreInstallationTemplates converts a list of v1alpha1 InstallationTemplates to their core version
func convertToCoreInstallationTemplates(installationTmpl []*lsv1alpha1.InstallationTemplate) ([]*core.InstallationTemplate, error) {
	coreInstTmpls := make([]*core.InstallationTemplate, len(installationTmpl))
	for i, tmpl := range installationTmpl {
		coreTmpl := &core.InstallationTemplate{}
		if err := lsv1alpha1.Convert_v1alpha1_InstallationTemplate_To_core_InstallationTemplate(tmpl, coreTmpl, nil); err != nil {
			return nil, err
		}
		coreInstTmpls[i] = coreTmpl
	}
	return coreInstTmpls, nil
}

// buildCoreImports converts the given list of versioned ImportDefinitions (potentially containing nested conditional imports) into a flattened list of core ImportDefinitions.
func (o *Operation) buildCoreImports(importList lsv1alpha1.ImportDefinitionList) ([]core.ImportDefinition, error) {
	coreImports := []core.ImportDefinition{}
	for _, importDef := range importList {
		coreImport := core.ImportDefinition{}
		if err := lsv1alpha1.Convert_v1alpha1_ImportDefinition_To_core_ImportDefinition(&importDef, &coreImport, nil); err != nil {
			return nil, err
		}
		coreImports = append(coreImports, coreImport)
		// recursively check for conditional imports
		_, ok := o.Inst.GetImports()[importDef.Name]
		if ok && len(importDef.ConditionalImports) > 0 {
			conditionalCoreImports, err := o.buildCoreImports(importDef.ConditionalImports)
			if err != nil {
				return nil, err
			}
			coreImports = append(coreImports, conditionalCoreImports...)
		}
	}
	return coreImports, nil
}

// getInstallationTemplate returns the installation template by name
func getInstallationTemplate(installationTmpl []*lsv1alpha1.InstallationTemplate, name string) (*lsv1alpha1.InstallationTemplate, bool) {
	for _, tmpl := range installationTmpl {
		if tmpl.Name == name {
			return tmpl, true
		}
	}
	return nil, false
}
