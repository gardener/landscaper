package landscaper

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
)

type BlueprintRenderer struct {
	cdList            *cdv2.ComponentDescriptorList
	componentResolver ctf.ComponentResolver
	repositoryContext *cdv2.UnstructuredTypedObject
}

type rendererTupleType struct {
	*cdv2.ComponentDescriptor
	*lsv1alpha1.Installation
	*blueprints.Blueprint
}

type ResolvedInstallation rendererTupleType
type RenderInput rendererTupleType

type RenderedDeployItemsSubInstallations struct {
	DeployItems               []*lsv1alpha1.DeployItem
	DeployItemTemplateState   map[string][]byte
	Installations             []ResolvedInstallation
	InstallationTemplateState map[string][]byte
}

func NewBlueprintRenderer(cdList *cdv2.ComponentDescriptorList, resolver ctf.ComponentResolver, repositoryContext *cdv2.UnstructuredTypedObject) *BlueprintRenderer {
	renderer := &BlueprintRenderer{
		cdList:            cdList,
		componentResolver: resolver,
		repositoryContext: repositoryContext,
	}
	return renderer
}

func (r *BlueprintRenderer) RenderDeployItemsAndSubInstallations(input *RenderInput, imports map[string]interface{}) (*RenderedDeployItemsSubInstallations, error) {
	if input == nil {
		return nil, fmt.Errorf("input may not be nil")
	}

	if input.Blueprint == nil {
		return nil, fmt.Errorf("blueprint may not be nil")
	}

	if err := r.validateImports(input, imports); err != nil {
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

func (r *BlueprintRenderer) RenderExportExecutions(input *RenderInput, installationDataImports, installationTargetImports, deployItemsExports map[string]interface{}) (map[string]interface{}, error) {
	var (
		blobResolver ctf.BlobResolver
		ctx          context.Context
	)

	ctx = context.Background()
	defer ctx.Done()

	if r.componentResolver != nil {
		var err error
		_, blobResolver, err = r.componentResolver.ResolveWithBlobResolver(ctx, r.getRepositoryContext(input), input.ComponentDescriptor.GetName(), input.ComponentDescriptor.GetVersion())
		if err != nil {
			return nil, fmt.Errorf("unable to get blob resolver: %w", err)
		}
	}

	values := map[string]interface{}{
		"deployitems": deployItemsExports,
		"dataobjects": installationDataImports,
		"targets":     installationTargetImports,
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	exports, err := template.New(gotemplate.New(blobResolver, templateStateHandler).WithInputFormatter(formatter), spiff.New(templateStateHandler).WithInputFormatter(formatter)).
		TemplateExportExecutions(input.Blueprint, values)

	if err != nil {
		return nil, fmt.Errorf("unable to template export executions: %w", err)
	}

	return exports, nil
}

func (r *BlueprintRenderer) renderDeployItems(input *RenderInput, imports map[string]interface{}) ([]*lsv1alpha1.DeployItem, map[string][]byte, error) {
	var (
		blobResolver ctf.BlobResolver
		ctx          context.Context
	)

	ctx = context.Background()
	defer ctx.Done()

	if input != nil && r.componentResolver != nil {
		var err error
		_, blobResolver, err = r.componentResolver.ResolveWithBlobResolver(ctx, r.getRepositoryContext(input), input.ComponentDescriptor.GetName(), input.ComponentDescriptor.GetVersion())
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get blob resolver: %w", err)
		}
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	deployItemTemplates, err := template.New(gotemplate.New(blobResolver, templateStateHandler).WithInputFormatter(formatter), spiff.New(templateStateHandler).WithInputFormatter(formatter)).
		TemplateDeployExecutions(template.DeployExecutionOptions{
			Imports:              imports,
			Blueprint:            input.Blueprint,
			ComponentDescriptor:  input.ComponentDescriptor,
			ComponentDescriptors: r.cdList,
			Installation:         input.Installation,
		})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to template deploy executions: %w", err)
	}

	deployItems := make([]*lsv1alpha1.DeployItem, len(deployItemTemplates))
	for i, tmpl := range deployItemTemplates {
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

func (r *BlueprintRenderer) renderSubInstallations(input *RenderInput, imports map[string]interface{}) ([]ResolvedInstallation, map[string][]byte, error) {
	ctx := context.Background()
	defer ctx.Done()

	installationTemplates, err := input.Blueprint.GetSubinstallations()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get subinstallation of blueprint: %w", err)
	}

	if len(installationTemplates) > 0 {
		installationTemplates, err = subinstallations.OrderInstallationTemplates(installationTemplates)
		if err != nil {
			return nil, nil, fmt.Errorf("unable for order subinstallations of blueprint: %w", err)
		}
	}

	templateStateHandler := template.NewMemoryStateHandler()
	formatter := template.NewTemplateInputFormatter(true)
	subInstallationTemplates, err := template.New(gotemplate.New(nil, templateStateHandler).WithInputFormatter(formatter), spiff.New(templateStateHandler).WithInputFormatter(formatter)).
		TemplateSubinstallationExecutions(template.DeployExecutionOptions{
			Imports:              imports,
			Blueprint:            input.Blueprint,
			ComponentDescriptor:  input.ComponentDescriptor,
			ComponentDescriptors: r.cdList,
			Installation:         input.Installation,
		})
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
			input.ComponentDescriptor,
			r.componentResolver,
			r.getRepositoryContext(input))
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get blueprint definition for subinstallation %q: %w", subInstTmpl.Name, err)
		}
		subInst.Spec.Blueprint = *subBlueprintDef
		subInst.Spec.ComponentDescriptor = subCd

		subBlueprint, err := blueprints.Resolve(ctx, r.componentResolver, subCd.Reference, *subBlueprintDef)
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

		cd, err := r.componentResolver.Resolve(ctx, r.getRepositoryContext(&RenderInput{
			ComponentDescriptor: input.ComponentDescriptor,
			Installation:        subInst,
		}), subComponentName, subComponentVersion)
		if err != nil {
			return nil, nil, err
		}

		subInstallations[i].ComponentDescriptor = cd
		subInstallations[i].Installation = subInst
		subInstallations[i].Blueprint = subBlueprint
	}

	return subInstallations, templateStateHandler, nil
}

func (r *BlueprintRenderer) validateImports(input *RenderInput, imports map[string]interface{}) error {

	validatorConfig := &jsonschema.ReferenceContext{
		LocalTypes:          input.Blueprint.Info.LocalTypes,
		BlueprintFs:         input.Blueprint.Fs,
		ComponentDescriptor: input.ComponentDescriptor,
		ComponentResolver:   r.componentResolver,
		RepositoryContext:   r.getRepositoryContext(input),
	}

	var allErr field.ErrorList
	for _, importDef := range input.Blueprint.Info.Imports {
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

		case lsv1alpha1.ImportTypeComponentDescriptor:
			allErr = append(allErr, validateComponentDescriptorImport(value, fldPath)...)

		case lsv1alpha1.ImportTypeComponentDescriptorList:
			allErr = append(allErr, validateComponentDescriptorListImport(value, fldPath)...)

		default:
			allErr = append(allErr, field.Invalid(fldPath, string(importDef.Type), "unknown import type"))
		}
	}

	return allErr.ToAggregate()
}

func (r *BlueprintRenderer) getRepositoryContext(input *RenderInput) *cdv2.UnstructuredTypedObject {
	if r.repositoryContext != nil {
		return r.repositoryContext
	}

	if input.Installation != nil && input.Installation.Spec.ComponentDescriptor != nil {
		if input.Installation.Spec.ComponentDescriptor.Reference != nil && input.Installation.Spec.ComponentDescriptor.Reference.RepositoryContext != nil {
			return input.Installation.Spec.ComponentDescriptor.Reference.RepositoryContext
		}
		if input.Installation.Spec.ComponentDescriptor.Inline != nil {
			return input.Installation.Spec.ComponentDescriptor.Inline.GetEffectiveRepositoryContext()
		}
	}

	if input.ComponentDescriptor != nil {
		return input.ComponentDescriptor.GetEffectiveRepositoryContext()
	}

	return nil
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

func validateComponentDescriptorImport(value interface{}, fldPath *field.Path) field.ErrorList {
	allErr := field.ErrorList{}
	_, ok := value.(map[string]interface{})
	if !ok {
		allErr = append(allErr, field.Invalid(fldPath, value, "a component descriptor is expected to be an object"))
		return allErr
	}

	return allErr
}

func validateComponentDescriptorListImport(value interface{}, fldPath *field.Path) field.ErrorList {
	allErr := field.ErrorList{}

	cdList, ok := value.([]interface{})
	if !ok {
		allErr = append(allErr, field.Invalid(fldPath, value, "a component descriptor list is expected to be a list"))
	}

	for i, cdObj := range cdList {
		allErr = append(allErr, validateComponentDescriptorImport(cdObj, fldPath.Index(i))...)
	}

	return allErr
}
