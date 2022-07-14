// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"context"
	"fmt"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"

	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// GetBlueprintDefinitionFromInstallationTemplate returns a Blueprint and a ComponentDescriptor from a subinstallation
func GetBlueprintDefinitionFromInstallationTemplate(
	inst *lsv1alpha1.Installation,
	subInstTmpl *lsv1alpha1.InstallationTemplate,
	cd *cdv2.ComponentDescriptor,
	compResolver ctf.ComponentResolver,
	repositoryContext *cdv2.UnstructuredTypedObject) (*lsv1alpha1.BlueprintDefinition, *lsv1alpha1.ComponentDescriptorDefinition, error) {
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
		_, res, err := uri.Get(cd, compResolver, repositoryContext)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve blueprint ref in component descriptor %s: %w", cd.Name, err)
		}
		// the result of the uri has to be an resource
		resource, ok := res.(cdv2.Resource)
		if !ok {
			return nil, nil, fmt.Errorf("expected a resource from the component descriptor %s", cd.Name)
		}

		subInstCompDesc, err := uri.GetComponent(cd, compResolver, repositoryContext)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve component of blueprint ref in component descriptor %s: %w", cd.Name, err)
		}

		// remove parent component descriptor
		cdDef = &lsv1alpha1.ComponentDescriptorDefinition{}

		// check, if the component descriptor of the subinstallation has been defined as a nested inline CD in the parent installation
		if inst.Spec.ComponentDescriptor != nil && inst.Spec.ComponentDescriptor.Inline != nil {
			for _, ref := range inst.Spec.ComponentDescriptor.Inline.ComponentReferences {
				if ref.ComponentName == subInstCompDesc.GetName() && ref.Version == subInstCompDesc.GetVersion() {
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
			latestRepoCtx := subInstCompDesc.GetEffectiveRepositoryContext()
			cdDef = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: latestRepoCtx,
					ComponentName:     subInstCompDesc.Name,
					Version:           subInstCompDesc.Version,
				},
			}
		}

		subBlueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resource.Name,
		}
	}

	return subBlueprint, cdDef, nil
}

// PseudoInstallation is a helper struct which can be easily constructed from Installations as well as from InstallationTemplates.
// It enables to use the same helper functions for both struct types.
type PseudoInstallation struct {
	Name               string
	Imports            *lsv1alpha1.InstallationImports
	ImportDataMappings *map[string]lsv1alpha1.AnyJSON
	Exports            *lsv1alpha1.InstallationExports
	ExportDataMappings *map[string]lsv1alpha1.AnyJSON
}

func AbstractInstallation(inst *lsv1alpha1.Installation) *PseudoInstallation {
	return &PseudoInstallation{
		Name:               inst.Name,
		Imports:            &inst.Spec.Imports,
		ImportDataMappings: &inst.Spec.ImportDataMappings,
		Exports:            &inst.Spec.Exports,
		ExportDataMappings: &inst.Spec.ExportDataMappings,
	}
}

func AbstractInstallations(insts []*lsv1alpha1.Installation) []*PseudoInstallation {
	res := make([]*PseudoInstallation, len(insts))
	for i := range insts {
		res[i] = AbstractInstallation(insts[i])
	}
	return res
}

func AbstractInstallationTemplate(tmpl *lsv1alpha1.InstallationTemplate) *PseudoInstallation {
	return &PseudoInstallation{
		Name:               tmpl.Name,
		Imports:            &tmpl.Imports,
		ImportDataMappings: &tmpl.ImportDataMappings,
		Exports:            &tmpl.Exports,
		ExportDataMappings: &tmpl.ExportDataMappings,
	}
}

func AbstractInstallationTemplates(tmpls []*lsv1alpha1.InstallationTemplate) []*PseudoInstallation {
	res := make([]*PseudoInstallation, len(tmpls))
	for i := range tmpls {
		res[i] = AbstractInstallationTemplate(tmpls[i])
	}
	return res
}

// ComputeInstallationDependencies computes the dependencies between the given subinstallations.
// It checks for every import of every subinstallation, whether that import is exported from another subinstallation.
// The resulting map contains one entry for every given subinstallation template, mapping that templates name to the
// set of templates (by name) it depends on.
func ComputeInstallationDependencies(installationTmpl []*PseudoInstallation) (map[string]sets.String, utils.ImportRelationships) {
	dataExports := map[string]string{}
	targetExports := map[string]string{}
	for _, tmpl := range installationTmpl {
		for _, exp := range tmpl.Exports.Data {
			dataExports[exp.DataRef] = tmpl.Name
		}
		for _, exp := range tmpl.Exports.Targets {
			targetExports[exp.Target] = tmpl.Name
		}
	}

	deps := map[string]sets.String{}
	impRels := utils.ImportRelationships{}
	for _, tmpl := range installationTmpl {
		tmp := sets.NewString()
		for _, imp := range tmpl.Imports.Data {
			if len(imp.DataRef) == 0 {
				// only dataRef imports can refer to sibling exports
				continue
			}
			source, ok := dataExports[imp.DataRef]
			if !ok {
				// no sibling exports this import, it has to come from the parent
				// this is already checked by validation, no need to verify it here
				continue
			}
			impRels.Add(source, tmpl.Name, imp.DataRef)
			tmp.Insert(source)
		}
		for _, imp := range tmpl.Imports.Targets {
			targets := []string{}
			if len(imp.Target) != 0 {
				targets = append(targets, imp.Target)
			} else if len(imp.Targets) != 0 {
				targets = imp.Targets
			} else {
				// targetListReferences can only refer to parent imports, not to sibling exports
				continue
			}
			for _, t := range targets {
				source, ok := targetExports[t]
				if !ok {
					// no sibling exports this import, it has to come from the parent
					// this is already checked by validation, no need to verify it here
					continue
				}
				impRels.Add(source, tmpl.Name, t)
				tmp.Insert(source)
			}
		}
		deps[tmpl.Name] = tmp
	}
	return deps, impRels
}

// OrderInstallationTemplates takes a list of installation templates and orders them according to their dependencies (as computed by ComputeInstallationDependencies).
// Installation templates which have been put into the new order are referred to as 'scheduled'.
func OrderInstallationTemplates(installationTmpl []*lsv1alpha1.InstallationTemplate) ([]*lsv1alpha1.InstallationTemplate, error) {
	// 'done' and 'ordered' more or less contain the same information:
	// which installation templates have been ordered yet.
	// 'ordered' is a list to preserve the computed order,
	// while 'done' is a set to efficiently check whether a specific installation template is already scheduled.
	ordered := []*lsv1alpha1.InstallationTemplate{}
	done := sets.NewString()
	dependencies, impRels := ComputeInstallationDependencies(AbstractInstallationTemplates(installationTmpl))
	// repeat until either
	// - all items from installationTmpl have been transferred to ordered (we are finished)
	// - no new items have been scheduled during the last run
	//   which hints that all items left are part of a cycle and the dependencies cannot be resolved
	for oldSize := -1; oldSize != len(done) && len(ordered) != len(installationTmpl); {
		oldSize = len(done)
		for _, tmpl := range installationTmpl {
			// skip if already scheduled
			if done.Has(tmpl.Name) {
				continue
			}
			// verify that all dependencies of the current subinstallation are already scheduled
			deps, ok := dependencies[tmpl.Name]
			if !ok {
				return nil, fmt.Errorf("subinstallation template %q not found in dependency hierarchy", tmpl.Name)
			}
			if len(deps) > len(done) {
				// installation template has more dependencies than are currently scheduled
				// so we don't have to check whether all of them are fulfilled because at least one won't be
				continue
			}
			canBeScheduled := true
			for d := range deps {
				if !done.Has(d) {
					// at least one dependency has not been scheduled yet
					canBeScheduled = false
					break
				}
			}
			// schedule subinstallation, if possible
			if canBeScheduled {
				ordered = append(ordered, tmpl)
				done.Insert(tmpl.Name)
			}
		}
	}
	if len(done) != len(installationTmpl) || len(ordered) != len(installationTmpl) { // these comparisons should be equivalent, just checking both to be sure ...
		// cyclic dependencies detected
		// identify installation templates which are part of the cycle
		missing := sets.NewString()
		for _, tmpl := range installationTmpl {
			if !done.Has(tmpl.Name) {
				missing.Insert(tmpl.Name)
			}
		}
		cycles := utils.DetermineCyclicDependencyDetails(missing, dependencies, impRels)
		return nil, newSubinstallationCyclicDependencyError("EnsureNestedInstallations", cycles)
	}
	return ordered, nil
}

// ValidateSubinstallations validates the installation templates in context of the current blueprint.
func (o *Operation) ValidateSubinstallations(installationTmpl []*lsv1alpha1.InstallationTemplate) error {
	coreInstTmpls, err := convertToCoreInstallationTemplates(installationTmpl)
	if err != nil {
		return err
	}

	coreImports, err := o.buildCoreImports(o.Inst.Blueprint.Info.Imports)
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
		_, ok := o.Inst.Imports[importDef.Name]
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

// CombinedPhase returns the combined phase of all subinstallations
func CombinedPhase(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation, subinsts ...*lsv1alpha1.Installation) (lsv1alpha1.ComponentInstallationPhase, error) {
	if len(subinsts) == 0 {
		var err error
		subinsts, err = installations.ListSubinstallations(ctx, kubeClient, inst)
		if err != nil {
			return "", err
		}
	}

	if len(subinsts) == 0 {
		return "", nil
	}

	phases := make([]lsv1alpha1.ComponentInstallationPhase, 0)
	for _, v := range subinsts {
		if v.Generation != v.Status.ObservedGeneration {
			phases = append(phases, lsv1alpha1.ComponentPhaseProgressing)
		} else {
			phases = append(phases, v.Status.Phase)
		}
	}

	return helper.CombinedInstallationPhase(phases...), nil
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

func newSubinstallationCyclicDependencyError(operation string, cycles []*utils.DependencyCycle) *lserrors.Error {
	var sb strings.Builder
	sb.WriteString("The following cyclic dependencies have been found in the nested installation templates: ")
	for i, c := range cycles {
		sb.WriteString("{")
		sb.WriteString(c.String())
		sb.WriteString("}")
		if i < len(cycles)-1 {
			sb.WriteString(", ")
		}
	}
	return lserrors.NewError(operation, "OrderNestedInstallationTemplates", sb.String(), lsv1alpha1.ErrorCyclicDependencies)
}
