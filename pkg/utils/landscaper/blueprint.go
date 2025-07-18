// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	lsblueprints "github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
	"github.com/gardener/landscaper/pkg/utils/dependencies"
)

// BlueprintRenderer is able to render a blueprint with given import values or exports for export templates.
type BlueprintRenderer struct {
	// cdList is the list of local component descriptors available to the renderer.
	cdList *model.ComponentVersionList
	// registryAccess is used to resolve component descriptors.
	registryAccess model.RegistryAccess
	// repositoryContext is an optional repository context used to overwrite the effective repository context of component descriptors.
	repositoryContext *types.UnstructuredTypedObject
}

// ResolvedInstallation contains a tuple of component descriptor, installation and blueprint.
type ResolvedInstallation struct {
	ComponentVersion model.ComponentVersion
	*lsv1alpha1.Installation
	*blueprints.Blueprint
}

// RenderedDeployItemsSubInstallations contains a list of rendered deployitems, deployitem state, installations and installation state.
type RenderedDeployItemsSubInstallations struct {
	// DeployItems contains the list of rendered deployitems.
	DeployItems []*lsv1alpha1.DeployItem
	// DeployItemTemplateState contains the rendered state of the deployitems templates.
	DeployItemTemplateState map[string][]byte
	// Installations contains the rendered installations.
	Installations []ResolvedInstallation
	// InstallationTemplateState contains the rendered state of the installation templates.
	InstallationTemplateState map[string][]byte
}

// NewBlueprintRenderer creates a new blueprint renderer. The arguments are optional and may be nil.
func NewBlueprintRenderer(cdList *model.ComponentVersionList, registryAccess model.RegistryAccess, repositoryContext *types.UnstructuredTypedObject) *BlueprintRenderer {
	renderer := &BlueprintRenderer{
		cdList:            cdList,
		registryAccess:    registryAccess,
		repositoryContext: repositoryContext,
	}
	return renderer
}

// RenderDeployItemsAndSubInstallations renders deploy items and subinstallations of a given blueprint using the given imports.
// The import values are validated with the JSON schemas defined in the blueprint.
func (r *BlueprintRenderer) RenderDeployItemsAndSubInstallations(input *ResolvedInstallation, imports map[string]interface{}) (*RenderedDeployItemsSubInstallations, error) {
	if input == nil {
		return nil, fmt.Errorf("render input may not be nil")
	}

	if input.Blueprint == nil {
		return nil, fmt.Errorf("blueprint may not be nil")
	}

	if err := r.validateImports(input, imports); err != nil {
		return nil, err
	}

	imports, err := r.RenderImportExecutions(input, imports)
	if err != nil {
		return nil, err
	}

	deployItems, deployItemsState, err := r.renderDeployItems(input, imports)
	if err != nil {
		return nil, err
	}

	subInstallations, subInstallationsState, err := r.renderSubInstallations(input, imports)
	if err != nil {
		return nil, err
	}

	renderOut := &RenderedDeployItemsSubInstallations{
		DeployItems:               deployItems,
		DeployItemTemplateState:   deployItemsState,
		Installations:             subInstallations,
		InstallationTemplateState: subInstallationsState,
	}
	return renderOut, nil
}

// RenderImportExecutions renders the export executions of the given blueprint and returns the rendered exports.
func (r *BlueprintRenderer) RenderImportExecutions(input *ResolvedInstallation, imports map[string]interface{}) (map[string]interface{}, error) {
	ctx := context.Background()
	defer ctx.Done()

	if input == nil {
		return nil, fmt.Errorf("render input may not be nil")
	}

	if input.Blueprint == nil {
		return nil, fmt.Errorf("blueprint may not be nil")
	}

	if len(input.Info.ImportExecutions) == 0 {
		// nothing to do if there aren't any ImportExecutions
		return imports, nil
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	tmpl := template.New(
		gotemplate.New(templateStateHandler, nil).WithInputFormatter(formatter),
		spiff.New(templateStateHandler, nil).WithInputFormatter(formatter))
	errorList, bindings, err := tmpl.TemplateImportExecutions(
		template.NewBlueprintExecutionOptions(
			input.Installation,
			input.Blueprint,
			input.ComponentVersion,
			r.cdList,
			imports))

	if err != nil {
		return nil, fmt.Errorf("unable to template export executions: %w", err)
	}

	if len(errorList) > 0 {
		return nil, fmt.Errorf("the following error(s) occurred in the import executions:\n\t%s", strings.Join(errorList, "\n\t"))
	}
	for k, v := range bindings {
		imports[k] = v
	}

	return imports, nil
}

// RenderExportExecutions renders the export executions of the given blueprint and returns the rendered exports.
func (r *BlueprintRenderer) RenderExportExecutions(input *ResolvedInstallation, imports, installationDataImports, installationTargetImports, deployItemsExports map[string]interface{}) (map[string]interface{}, error) {
	ctx := context.Background()
	defer ctx.Done()

	if input == nil {
		return nil, fmt.Errorf("render input may not be nil")
	}

	if input.Blueprint == nil {
		return nil, fmt.Errorf("blueprint may not be nil")
	}

	values := map[string]interface{}{
		"deployitems": deployItemsExports,
		"dataobjects": installationDataImports,
		"targets":     installationTargetImports,
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	tmpl := template.New(
		gotemplate.New(templateStateHandler, nil).WithInputFormatter(formatter),
		spiff.New(templateStateHandler, nil).WithInputFormatter(formatter))
	exports, err := tmpl.TemplateExportExecutions(
		template.NewExportExecutionOptions(
			template.NewBlueprintExecutionOptions(
				input.Installation,
				input.Blueprint,
				input.ComponentVersion,
				r.cdList,
				imports), values))

	if err != nil {
		return nil, fmt.Errorf("unable to template export executions: %w", err)
	}

	return exports, nil
}

// renderDeployItems renders deploy items.
func (r *BlueprintRenderer) renderDeployItems(input *ResolvedInstallation, imports map[string]interface{}) ([]*lsv1alpha1.DeployItem, map[string][]byte, error) {
	ctx := context.Background()
	defer ctx.Done()

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	tmpl := template.New(
		gotemplate.New(templateStateHandler, nil).WithInputFormatter(formatter),
		spiff.New(templateStateHandler, nil).WithInputFormatter(formatter))
	executions, err := tmpl.TemplateDeployExecutions(
		template.NewDeployExecutionOptions(
			template.NewBlueprintExecutionOptions(
				input.Installation,
				input.Blueprint,
				input.ComponentVersion,
				r.cdList,
				imports)))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to template deploy executions: %w", err)
	}

	// map deployitem specifications into templates for executions
	// includes resolving target import references to target object references
	deployItemTemplates := make(core.DeployItemTemplateList, len(executions))
	for i, elem := range executions {
		var target *core.ObjectReference
		if elem.Target != nil {
			target = &core.ObjectReference{
				Name:      elem.Target.Name,
				Namespace: input.Namespace,
			}
			if elem.Target.Index != nil {
				// targetlist import reference
				raw := imports[elem.Target.Import]
				imp := input.GetImportByName(elem.Target.Import)
				if imp == nil {
					return nil, nil, deployItemSpecificationError(elem.Name, "targetlist import %q not found", elem.Target.Import)
				}
				if imp.Type != lsv1alpha1.ImportTypeTargetList {
					return nil, nil, deployItemSpecificationError(elem.Name, "import %q is not a targetlist", elem.Target.Import)
				}
				if raw == nil {
					return nil, nil, deployItemSpecificationError(elem.Name, "no value for import %q given", elem.Target.Import)
				}
				val, ok := raw.([]map[string]interface{})
				if !ok {
					return nil, nil, deployItemSpecificationError(elem.Name, "invalid target spec for import %q", elem.Target.Import)
				}
				if *elem.Target.Index < 0 || *elem.Target.Index >= len(val) {
					return nil, nil, deployItemSpecificationError(elem.Name, "index %d out of bounds", *elem.Target.Index)
				}
				name, _, err := unstructured.NestedString(val[*elem.Target.Index], "metadata", "name")
				if err != nil {
					return nil, nil, err
				}
				namespace, _, _ := unstructured.NestedString(val[*elem.Target.Index], "metadata", "namespace")
				target.Name = name
				target.Namespace = namespace
			} else if len(elem.Target.Import) > 0 {
				// single target import reference
				raw := imports[elem.Target.Import]
				imp := input.GetImportByName(elem.Target.Import)
				if imp == nil {
					return nil, nil, deployItemSpecificationError(elem.Name, "target import %q not found", elem.Target.Import)
				}
				if imp.Type != lsv1alpha1.ImportTypeTarget {
					return nil, nil, deployItemSpecificationError(elem.Name, "import %q is not a target", elem.Target.Import)
				}
				if raw == nil {
					return nil, nil, deployItemSpecificationError(elem.Name, "no value for import %q given", elem.Target.Import)
				}
				val, ok := raw.(map[string]interface{})
				if !ok {
					return nil, nil, deployItemSpecificationError(elem.Name, "invalid target spec for import %q", elem.Target.Import)
				}
				name, _, err := unstructured.NestedString(val, "metadata", "name")
				if err != nil {
					return nil, nil, err
				}
				namespace, _, _ := unstructured.NestedString(val, "metadata", "namespace")
				target.Name = name
				target.Namespace = namespace
			} else if len(elem.Target.Name) == 0 {
				return nil, nil, deployItemSpecificationError(elem.Name, "empty target reference")
			}
		}

		deployItemTemplates[i] = core.DeployItemTemplate{
			Name:          elem.Name,
			Type:          elem.Type,
			Target:        target,
			Labels:        elem.Labels,
			Configuration: elem.Configuration,
			DependsOn:     elem.DependsOn,
		}
	}

	versionedDeployItemTemplateList := lsv1alpha1.DeployItemTemplateList{}
	if err := lsv1alpha1.Convert_core_DeployItemTemplateList_To_v1alpha1_DeployItemTemplateList(&deployItemTemplates, &versionedDeployItemTemplateList, nil); err != nil {
		return nil, nil, fmt.Errorf("error converting internal representation of deployitem templates to versioned one: %w", err)
	}

	deployItems := make([]*lsv1alpha1.DeployItem, len(versionedDeployItemTemplateList))
	for i, tmpl := range versionedDeployItemTemplateList {
		di := &lsv1alpha1.DeployItem{}
		if err := kutil.InjectTypeInformation(di, api.LandscaperScheme); err != nil {
			return nil, nil, fmt.Errorf("unable to inject deploy item type information for %q: %w", tmpl.Name, err)
		}
		execution.ApplyDeployItemTemplate(di, tmpl)
		di.Name = tmpl.Name
		deployItems[i] = di
	}
	return deployItems, templateStateHandler, nil
}

// renderSubInstallations renders subinstallations.
func (r *BlueprintRenderer) renderSubInstallations(input *ResolvedInstallation, imports map[string]interface{}) ([]ResolvedInstallation, map[string][]byte, error) {
	ctx := context.Background()
	defer ctx.Done()

	inputRepositoryContext, err := r.getRepositoryContext(input)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get repository context for input installation: %w", err)
	}

	installationTemplates, err := input.GetSubinstallations()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get subinstallation of blueprint: %w", err)
	}

	if len(installationTemplates) > 0 {
		installationTemplates, err = dependencies.CheckForCyclesAndDuplicateExports(installationTemplates, true)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to order subinstallations of blueprint: %w", err)
		}
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	tmpl := template.New(
		gotemplate.New(templateStateHandler, nil).WithInputFormatter(formatter),
		spiff.New(templateStateHandler, nil).WithInputFormatter(formatter))
	subInstallationTemplates, err := tmpl.TemplateSubinstallationExecutions(
		template.NewDeployExecutionOptions(
			template.NewBlueprintExecutionOptions(
				input.Installation,
				input.Blueprint,
				input.ComponentVersion,
				r.cdList,
				imports)))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to template subinstalltion executions: %w", err)
	}

	installationTemplates = append(installationTemplates, subInstallationTemplates...)
	subInstallations := make([]ResolvedInstallation, len(installationTemplates))
	for i, subInstTmpl := range installationTemplates {
		subInst := &lsv1alpha1.Installation{}
		subInst.Name = subInstTmpl.Name
		subInst.Spec = lsv1alpha1.InstallationSpec{
			Imports:            subInstTmpl.Imports,
			ImportDataMappings: subInstTmpl.ImportDataMappings,
			Exports:            subInstTmpl.Exports,
			ExportDataMappings: subInstTmpl.ExportDataMappings,
		}
		subBlueprintDef, subCd, err := subinstallations.GetBlueprintDefinitionFromInstallationTemplate(
			input.Installation,
			subInstTmpl,
			input.ComponentVersion,
			inputRepositoryContext,
			nil)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get blueprint definition for subinstallation %q: %w", subInstTmpl.Name, err)
		}
		subInst.Spec.Blueprint = *subBlueprintDef
		subInst.Spec.ComponentDescriptor = subCd

		subInstRepositoryContext, err := r.getRepositoryContext(&ResolvedInstallation{
			ComponentVersion: input.ComponentVersion,
			Installation:     subInst,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get repository context for subinstallation %q: %w", subInstTmpl.Name, err)
		}

		subCd.Reference.RepositoryContext = subInstRepositoryContext
		subBlueprint, err := lsblueprints.Resolve(ctx, r.registryAccess, subCd.Reference, *subBlueprintDef, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to resolve blueprint for subinstallation %q: %w", subInstTmpl.Name, err)
		}

		var (
			subComponentName, subComponentVersion string
		)

		if subInst.Spec.ComponentDescriptor.Reference != nil {
			subComponentName = subInst.Spec.ComponentDescriptor.Reference.ComponentName
			subComponentVersion = subInst.Spec.ComponentDescriptor.Reference.Version
		} else if subInst.Spec.ComponentDescriptor.Inline != nil {
			subComponentName = subInst.Spec.ComponentDescriptor.Inline.Name
			subComponentVersion = subInst.Spec.ComponentDescriptor.Inline.Version
		}

		subComponentDescriptor, err := r.registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: subInstRepositoryContext,
			ComponentName:     subComponentName,
			Version:           subComponentVersion,
		})
		if err != nil {
			return nil, nil, err
		}

		subInstallations[i].ComponentVersion = subComponentDescriptor
		subInstallations[i].Installation = subInst
		subInstallations[i].Blueprint = subBlueprint
	}

	return subInstallations, templateStateHandler, nil
}

// validateImports validates the imports with the JSON schemas defined in the blueprint
func (r *BlueprintRenderer) validateImports(input *ResolvedInstallation, imports map[string]interface{}) error {

	inputRepositoryContext, err := r.getRepositoryContext(input)
	if err != nil {
		return fmt.Errorf("unable to get repository context during validation of imports: %w", err)
	}

	validatorConfig := &jsonschema.ReferenceContext{
		LocalTypes:        input.Info.LocalTypes,
		BlueprintFs:       input.Fs,
		ComponentVersion:  input.ComponentVersion,
		RegistryAccess:    r.registryAccess,
		RepositoryContext: inputRepositoryContext,
	}

	var allErr field.ErrorList
	for _, importDef := range input.Info.Imports {
		fldPath := field.NewPath(importDef.Name)
		value, ok := imports[importDef.Name]
		if !ok {
			if *importDef.Required {
				allErr = append(allErr, field.Required(fldPath, "Import is required"))
			}
			continue
		}
		switch importDef.Type {
		case lsv1alpha1.ImportTypeData:
			if err := jsonschema.ValidateGoStruct(importDef.Schema.RawMessage, value, validatorConfig); err != nil {
				allErr = append(allErr, field.Invalid(
					fldPath,
					value,
					fmt.Sprintf("invalid imported value: %s", err.Error())))
			}
		case lsv1alpha1.ImportTypeTarget:
			allErr = append(allErr, validateTargetImport(value, importDef.TargetType, fldPath)...)

		case lsv1alpha1.ImportTypeTargetList:
			allErr = append(allErr, validateTargetListImport(value, importDef.TargetType, fldPath)...)

		case lsv1alpha1.ImportTypeTargetMap:
			allErr = append(allErr, validateTargetMapImport(value, importDef.TargetType, fldPath)...)

		default:
			allErr = append(allErr, field.Invalid(fldPath, string(importDef.Type), "unknown import type"))
		}
	}

	return allErr.ToAggregate()
}

// getRepositoryContext retrieves the correct repository context.
// The priority is as following:
// 1. explicitly user defined repository context
// 2. repository context defined in the installation
// 3. effective repository context defined in the component descriptor.
func (r *BlueprintRenderer) getRepositoryContext(input *ResolvedInstallation) (*types.UnstructuredTypedObject, error) {
	if r.repositoryContext != nil {
		return r.repositoryContext, nil
	}

	if input.Installation != nil && input.Spec.ComponentDescriptor != nil {
		if input.Spec.ComponentDescriptor.Reference != nil && input.Spec.ComponentDescriptor.Reference.RepositoryContext != nil {
			return input.Spec.ComponentDescriptor.Reference.RepositoryContext, nil
		}
		if input.Spec.ComponentDescriptor.Inline != nil {
			return input.Spec.ComponentDescriptor.Inline.GetEffectiveRepositoryContext(), nil
		}
	}

	if input.ComponentVersion != nil {
		repositoryContext := input.ComponentVersion.GetRepositoryContext()

		return repositoryContext, nil
	}

	return nil, nil
}

func validateTargetImport(value interface{}, expectedTargetType string, fldPath *field.Path) field.ErrorList {
	allErr := field.ErrorList{}

	targetObj, ok := value.(map[string]interface{})
	if !ok {
		allErr = append(allErr, field.Invalid(fldPath, value, "a target is expected to be an object"))
		return allErr
	}
	targetType, _, err := unstructured.NestedString(targetObj, "spec", "type")
	if err != nil {
		allErr = append(allErr, field.Invalid(
			fldPath,
			value,
			fmt.Sprintf("unable to get type of target: %s", err.Error())))
		return allErr
	}
	_, _, err = unstructured.NestedString(targetObj, "metadata", "name")
	if err != nil {
		allErr = append(allErr, field.Invalid(
			fldPath,
			value,
			fmt.Sprintf("unable to get name of target: %s", err.Error())))
		return allErr
	}
	if targetType != expectedTargetType {
		allErr = append(allErr, field.Invalid(
			fldPath,
			targetType,
			fmt.Sprintf("expected target type to be %q but got %q", expectedTargetType, targetType)))
		return allErr
	}

	return allErr
}

func validateTargetListImport(value interface{}, expectedTargetType string, fldPath *field.Path) field.ErrorList {
	allErr := field.ErrorList{}

	targetList, ok := value.([]interface{})
	if !ok {
		allErr = append(allErr, field.Invalid(fldPath, value, "a target list is expected to be a list"))
	}

	for i, targetObj := range targetList {
		allErr = append(allErr, validateTargetImport(targetObj, expectedTargetType, fldPath.Index(i))...)
	}

	return allErr
}

func validateTargetMapImport(value interface{}, expectedTargetType string, fldPath *field.Path) field.ErrorList {
	allErr := field.ErrorList{}

	targetMap, ok := value.(map[string]interface{})
	if !ok {
		allErr = append(allErr, field.Invalid(fldPath, value, "a target map is expected to be a map"))
	}

	for targetMapKey, targetObj := range targetMap {
		allErr = append(allErr, validateTargetImport(targetObj, expectedTargetType, fldPath.Key(targetMapKey))...)
	}

	return allErr
}

func deployItemSpecificationError(name, message string, args ...interface{}) error {
	return fmt.Errorf(fmt.Sprintf("invalid deployitem specification %q: ", name)+message, args...)
}
