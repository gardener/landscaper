// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper

import (
	"fmt"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
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
	"github.com/gardener/landscaper/pkg/utils"
)

type BlueprintRenderArgs struct {
	Fs            vfs.FileSystem
	BlueprintPath string
	// ImportValuesFilepath is a path to to a imports.yaml file
	// +optional
	ImportValuesFilepath string
	// Imports describes the imports that should be used for the blueprint templating.
	// If the imports values filepath is given and the imports then both imports are merged.
	// +optional
	Imports *Imports
	// ComponentDescriptorFilepath describes the path to the component descriptor.
	ComponentDescriptorFilepath string
	// ComponentDescriptorList is a list of component descriptors that should be the transitive component references of the original component descriptor.
	ComponentDescriptorList *cdv2.ComponentDescriptorList
	// ComponentResolver implements a component descriptor resolver.
	ComponentResolver ctf.ComponentResolver

	// RootDir describes a directory that is used to default the other filepaths.
	// The blueprint is expected to have the following structure
	// <root Dir>
	// - blueprint
	//   - blueprint.yaml
	// - examples
	//   - imports.yaml
	//   - component-descriptor.yaml
	// +optional
	RootDir string
}

// Default defaults the BlueprintRender args
func (args *BlueprintRenderArgs) Default() error {
	if args.Fs == nil {
		args.Fs = osfs.New()
	}
	if len(args.RootDir) == 0 {
		args.RootDir = "../"
	}
	exampleDir := filepath.Join(args.RootDir, "example")
	if len(args.BlueprintPath) == 0 {
		args.BlueprintPath = filepath.Join(args.RootDir, "blueprint")
		if _, err := args.Fs.Stat(args.BlueprintPath); err != nil {
			args.BlueprintPath = ""
		}
	}
	if args.Imports == nil && len(args.ImportValuesFilepath) == 0 {
		args.ImportValuesFilepath = filepath.Join(exampleDir, "imports.yaml")
		if _, err := args.Fs.Stat(args.ImportValuesFilepath); err != nil {
			args.ImportValuesFilepath = ""
		}
	}
	if len(args.ComponentDescriptorFilepath) == 0 {
		args.ComponentDescriptorFilepath = filepath.Join(exampleDir, "component-descriptor.yaml")
		if _, err := args.Fs.Stat(args.ComponentDescriptorFilepath); err != nil {
			args.ComponentDescriptorFilepath = ""
		}
	}
	if args.ComponentDescriptorList == nil {
		args.ComponentDescriptorList = &cdv2.ComponentDescriptorList{}
	}
	return nil
}

// BlueprintRenderOut describes the output of the blueprint render function.
type BlueprintRenderOut struct {
	DeployItems               []*lsv1alpha1.DeployItem
	DeployItemTemplateState   map[string][]byte
	Installations             []*lsv1alpha1.Installation
	InstallationTemplateState map[string][]byte
}

// Imports describes the json/yaml file format for blueprint render imports.
type Imports struct {
	Imports map[string]interface{} `json:"imports"`
}

// RenderBlueprint renders a blueprint
func RenderBlueprint(args BlueprintRenderArgs) (*BlueprintRenderOut, error) {
	if err := args.Default(); err != nil {
		return nil, fmt.Errorf("unable to default args: %w", err)
	}
	imports := Imports{}
	if len(args.ImportValuesFilepath) != 0 {
		if err := utils.YAMLReadFromFile(args.Fs, args.ImportValuesFilepath, &imports); err != nil {
			return nil, fmt.Errorf("unable to read imports from %q: %w", args.ImportValuesFilepath, err)
		}
	}

	// merge imports
	if args.Imports != nil {
		MergeImports(&imports, args.Imports)
	}

	var cd *cdv2.ComponentDescriptor
	if len(args.ComponentDescriptorFilepath) != 0 {
		cd = &cdv2.ComponentDescriptor{}
		data, err := vfs.ReadFile(args.Fs, args.ComponentDescriptorFilepath)
		if err != nil {
			return nil, fmt.Errorf("unable to read component descriptor from %q: %w", args.ComponentDescriptorFilepath, err)
		}
		if err := codec.Decode(data, cd); err != nil {
			return nil, fmt.Errorf("unable to decode component descriptor from %q: %w", args.ComponentDescriptorFilepath, err)
		}
	}

	bpFs, err := projectionfs.New(args.Fs, args.BlueprintPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create blueprint fs for %q: %w", args.BlueprintPath, err)
	}
	blueprint, err := blueprints.NewFromFs(bpFs)
	if err != nil {
		return nil, fmt.Errorf("unable to read blueprint from %q: %w", args.BlueprintPath, err)
	}

	if err := ValidateImports(blueprint, &imports, cd, args.ComponentResolver); err != nil {
		return nil, err
	}

	sampleRepository, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/components", ""))
	if err != nil {
		return nil, fmt.Errorf("unable to parse sample repository context: %w", err)
	}
	inst := &lsv1alpha1.Installation{}
	inst.Spec.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
		ResourceName: "example-blueprint",
	}
	inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &sampleRepository,
			ComponentName:     "my-example-component",
			Version:           "v0.0.0",
		},
	}
	if cd != nil {
		inst.Spec.ComponentDescriptor.Reference.ComponentName = cd.GetName()
		inst.Spec.ComponentDescriptor.Reference.Version = cd.GetVersion()
		if len(cd.RepositoryContexts) != 0 {
			repoCtx := cd.GetEffectiveRepositoryContext()
			inst.Spec.ComponentDescriptor.Reference.RepositoryContext = repoCtx
		}
	}

	deployItems, deployItemsState, err := RenderBlueprintDeployItems(
		blueprint,
		imports,
		cd,
		args.ComponentDescriptorList,
		inst)
	if err != nil {
		return nil, err
	}

	installations, installationsState, err := RenderBlueprintSubInstallations(
		blueprint,
		imports,
		cd,
		args.ComponentDescriptorList,
		args.ComponentResolver,
		inst)
	if err != nil {
		return nil, err
	}

	return &BlueprintRenderOut{
		DeployItems:               deployItems,
		DeployItemTemplateState:   deployItemsState,
		Installations:             installations,
		InstallationTemplateState: installationsState,
	}, nil
}

func RenderBlueprintDeployItems(
	blueprint *blueprints.Blueprint,
	imports Imports,
	cd *cdv2.ComponentDescriptor,
	cdList *cdv2.ComponentDescriptorList,
	inst *lsv1alpha1.Installation) ([]*lsv1alpha1.DeployItem, map[string][]byte, error) {

	templateStateHandler := template.NewMemoryStateHandler()
	deployItemTemplates, err := template.New(gotemplate.New(nil, templateStateHandler), spiff.New(templateStateHandler)).
		TemplateDeployExecutions(template.DeployExecutionOptions{
			Imports:              imports.Imports,
			Blueprint:            blueprint,
			ComponentDescriptor:  cd,
			ComponentDescriptors: cdList,
			Installation:         inst,
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

func RenderBlueprintSubInstallations(
	blueprint *blueprints.Blueprint,
	imports Imports,
	cd *cdv2.ComponentDescriptor,
	cdList *cdv2.ComponentDescriptorList,
	compResolver ctf.ComponentResolver,
	inst *lsv1alpha1.Installation) ([]*lsv1alpha1.Installation, map[string][]byte, error) {

	installationTemplates, err := blueprint.GetSubinstallations()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get subinstallation of blueprint: %w", err)
	}

	templateStateHandler := template.NewMemoryStateHandler()
	subInstallationTemplates, err := template.New(gotemplate.New(nil, templateStateHandler), spiff.New(templateStateHandler)).
		TemplateSubinstallationExecutions(template.DeployExecutionOptions{
			Imports:              imports.Imports,
			Blueprint:            blueprint,
			ComponentDescriptor:  cd,
			ComponentDescriptors: cdList,
			Installation:         inst,
		})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to template subinstalltion executions: %w", err)
	}

	installationTemplates = append(installationTemplates, subInstallationTemplates...)
	installations := make([]*lsv1alpha1.Installation, len(installationTemplates))
	for i, subInstTmpl := range installationTemplates {
		subInst := &lsv1alpha1.Installation{}
		subInst.Name = subInstTmpl.Name
		subInst.Spec = lsv1alpha1.InstallationSpec{
			Imports:            subInstTmpl.Imports,
			ImportDataMappings: subInstTmpl.ImportDataMappings,
			Exports:            subInstTmpl.Exports,
			ExportDataMappings: subInstTmpl.ExportDataMappings,
		}
		subBlueprint, _, err := subinstallations.GetBlueprintDefinitionFromInstallationTemplate(
			inst,
			subInstTmpl,
			cd,
			compResolver)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get blueprint for subinstallation %q: %w", subInstTmpl.Name, err)
		}
		subInst.Spec.Blueprint = *subBlueprint
		installations[i] = subInst
	}

	return installations, templateStateHandler, nil
}

// ValidateImports the imports for a blueprint.
func ValidateImports(bp *blueprints.Blueprint,
	imports *Imports,
	cd *cdv2.ComponentDescriptor,
	componentResolver ctf.ComponentResolver) error {

	validatorConfig := &jsonschema.ReferenceContext{
		LocalTypes:          bp.Info.LocalTypes,
		BlueprintFs:         bp.Fs,
		ComponentDescriptor: cd,
		ComponentResolver:   componentResolver,
	}

	var allErr field.ErrorList
	for _, importDef := range bp.Info.Imports {
		fldPath := field.NewPath(importDef.Name)
		value, ok := imports.Imports[importDef.Name]
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

// MergeImports merges all imports of b into a.
func MergeImports(a, b *Imports) {
	if a.Imports == nil {
		a.Imports = b.Imports
		return
	}
	for key, val := range b.Imports {
		a.Imports[key] = val
	}
}
