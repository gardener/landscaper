// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package landscaper

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	gotmpl "text/template"

	"github.com/Masterminds/sprig/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
)

const (
	// rootInstallationName is the name of the auto-generated root installation
	rootInstallationName = "root"
	// rootInstallationAnnotation identifies the auto-generated root installation
	rootInstallationAnnotation = "landscaper.gardener.cloud/simulator-root-installation"
)

// InstallationPath defines elements in an installation, subinstallation chain
type InstallationPath struct {
	// name is the name of the current installation in the path
	name string
	// parent is the parent installation of the current path element.
	parent *InstallationPath
}

// NewInstallationPath creates a new installation path with the root installation named after given name.
func NewInstallationPath(name string) *InstallationPath {
	return &InstallationPath{
		name:   name,
		parent: nil,
	}
}

// Child creates a new child installation element with the given name.
func (p *InstallationPath) Child(name string) *InstallationPath {
	return &InstallationPath{
		name:   name,
		parent: p,
	}
}

// String converts the installation path into a file system path representation,
// starting at the root installation.
func (p *InstallationPath) String() string {
	var elems []string

	currPath := p
	for ; currPath != nil; currPath = currPath.parent {
		elems = append(elems, currPath.name)
	}

	// reverse the ordering, path starts with a root element and resembles a file system path
	for i, j := 0, len(elems)-1; i < j; i, j = i+1, j-1 {
		elems[i], elems[j] = elems[j], elems[i]
	}

	return path.Join(elems...)
}

// ExportTemplates contains a list of deploy item export templates.
type ExportTemplates struct {
	// DeployItemExports is a list of export templates that are matched with deploy items of installations.
	// The template has to output a valid yaml structure that contains a map under the key "exports".
	// Input parameters for the template are:
	// "imports": the installation imports
	//	"installationPath": the complete installation path that contains the deploy item, which is also used for selecting the template
	//	"templateName": the user specified name of this template
	//	"deployItem": the complete deploy item structure
	//  "cd": the component descriptor
	//  "components": the component descriptor list
	DeployItemExports []*ExportTemplate `json:"deployItems"`
	// InstallationExports is a list of export templates that are matched with installations.
	// The template has to output a valid yaml structure that contains a map under the key "dataExports" and the key "targetExports".
	// Input parameters for the template are:
	// "imports": the installation imports
	//	"installationPath": the complete path to the installation, which is also used for selecting the template
	//	"templateName": the user specified name of this template
	//	"installation": the complete installation structure
	//  "cd": the component descriptor
	//  "components": the component descriptor list
	//	"state": contains the calculated installation state
	InstallationExports []*ExportTemplate `json:"installations"`
}

// ExportTemplate contains a template definition that is executed once the selector matches.
type ExportTemplate struct {
	// Name is the name of the export template.
	Name string `json:"name"`
	// Selector; for deploy items: is a regular expression that must match the installation path and deploy item name.
	// Example: "root/installationA/.*/myDeployItem.*
	// for installations: is a regular expression that must match the installation path
	// Example: ".*/installationA"
	Selector string `json:"selector"`
	// Template contains the go template that must output a valid yaml structure.
	Template string `json:"template"`

	// SelectorRegexp is the compiled regular expression of the selector.
	SelectorRegexp *regexp.Regexp `json:"-"`
}

// InstallationSimulatorCallbacks are called when installations, deploy items, imports, exports or state elements are found
// during the simulation run.
type InstallationSimulatorCallbacks interface {
	// OnInstallation is called when a new installation was found.
	OnInstallation(path string, installation *lsv1alpha1.Installation)
	// OnInstallationTemplateState is called when a new installation template state was found.
	OnInstallationTemplateState(path string, state map[string][]byte)
	// OnImports is called when imports of an installation are found.
	OnImports(path string, imports map[string]interface{})
	// OnDeployItem is called when a new deploy item was found.
	OnDeployItem(path string, deployItem *lsv1alpha1.DeployItem)
	// OnDeployItemTemplateState is called when a new deploy item template state was found.
	OnDeployItemTemplateState(path string, state map[string][]byte)
	// OnExports is called when exports of an installation are found.
	OnExports(path string, exports map[string]interface{})
}

// emptySimulatorCallbacks are empty simulator callbacks that are used when no user defined callbacks are set.
type emptySimulatorCallbacks struct {
}

func (c emptySimulatorCallbacks) OnInstallation(_ string, _ *lsv1alpha1.Installation)       {}
func (c emptySimulatorCallbacks) OnInstallationTemplateState(_ string, _ map[string][]byte) {}
func (c emptySimulatorCallbacks) OnImports(_ string, _ map[string]interface{})              {}
func (c emptySimulatorCallbacks) OnDeployItem(_ string, _ *lsv1alpha1.DeployItem)           {}
func (c emptySimulatorCallbacks) OnDeployItemTemplateState(_ string, _ map[string][]byte)   {}
func (c emptySimulatorCallbacks) OnExports(_ string, _ map[string]interface{})              {}

// Exports contains exported data objects and targets.
type Exports struct {
	// DataObjects contains data object exports.
	DataObjects map[string]interface{}
	// Targets contains target exports.
	Targets map[string]interface{}
}

// InstallationExports contains data exported by an installation.
type InstallationExports Exports

// BlueprintExports contains data exported by a blueprint.
type BlueprintExports Exports

// InstallationSimulator simulations the landscaper handling of installations with its deploy items and subinstallations.
// The exports of deploy items are simulated via user defined ExportTemplates.
type InstallationSimulator struct {
	// blueprintRenderer is used to render blueprints of installations.
	blueprintRenderer *BlueprintRenderer
	// exportTemplates contains the user defined export templates.
	exportTemplates ExportTemplates
	// callbacks contains the user callbacks.
	callbacks InstallationSimulatorCallbacks
}

// NewInstallationSimulator creates a new installation simulator.
// The repositoryContext parameter is optional and can be set to nil.
func NewInstallationSimulator(cdList *cdv2.ComponentDescriptorList,
	resolver ctf.ComponentResolver,
	repositoryContext *cdv2.UnstructuredTypedObject,
	exportTemplates ExportTemplates) (*InstallationSimulator, error) {

	for _, template := range exportTemplates.DeployItemExports {
		var err error
		template.SelectorRegexp, err = regexp.Compile(template.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to compile deploy item export selector %s: %w", template.Name, err)
		}
	}

	for _, template := range exportTemplates.InstallationExports {
		var err error
		template.SelectorRegexp, err = regexp.Compile(template.Selector)
		if err != nil {
			return nil, fmt.Errorf("failed to compile installation export selector %s: %w", template.Name, err)
		}
	}

	return &InstallationSimulator{
		blueprintRenderer: NewBlueprintRenderer(cdList, resolver, repositoryContext),
		exportTemplates:   exportTemplates,
		callbacks:         emptySimulatorCallbacks{},
	}, nil
}

// SetCallbacks sets user defined simulator callbacks.
func (s *InstallationSimulator) SetCallbacks(callbacks InstallationSimulatorCallbacks) *InstallationSimulator {
	s.callbacks = callbacks
	return s
}

// Run starts the simulation for the given component descriptor, blueprint and imports and returns the calculated exports.
func (s *InstallationSimulator) Run(cd *cdv2.ComponentDescriptor, blueprint *blueprints.Blueprint, dataImports, targetImports map[string]interface{}) (*BlueprintExports, error) {
	ctx := &ResolvedInstallation{
		ComponentDescriptor: cd,
		Installation: &lsv1alpha1.Installation{
			ObjectMeta: metav1.ObjectMeta{
				Name: rootInstallationName,
				Annotations: map[string]string{
					rootInstallationAnnotation: rootInstallationName,
				},
			},
		},
		Blueprint: blueprint,
	}

	_, exports, err := s.executeInstallation(ctx, nil, dataImports, targetImports)
	return exports, err
}

// executeInstallation calculates the exports of the current installation and calls itself recursively for its subinstallations.
func (s *InstallationSimulator) executeInstallation(ctx *ResolvedInstallation, installationPath *InstallationPath, dataImports, targetImports map[string]interface{}) (*InstallationExports, *BlueprintExports, error) {
	if installationPath == nil {
		installationPath = NewInstallationPath(ctx.Installation.Name)
	} else {
		installationPath = installationPath.Child(ctx.Installation.Name)
	}

	pathString := installationPath.String()

	imports := make(map[string]interface{})
	mergeMaps(imports, dataImports)
	mergeMaps(imports, targetImports)

	s.callbacks.OnInstallation(pathString, ctx.Installation)
	s.callbacks.OnImports(pathString, imports)

	renderedDeployItemsAndSubInst, err := s.blueprintRenderer.RenderDeployItemsAndSubInstallations(ctx, imports)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render deploy items and subinstallations %q: %w", pathString, err)
	}

	if len(renderedDeployItemsAndSubInst.InstallationTemplateState) > 0 {
		s.callbacks.OnInstallationTemplateState(pathString, renderedDeployItemsAndSubInst.InstallationTemplateState)
	}

	exportsByDeployItem, err := s.handleDeployItems(pathString, renderedDeployItemsAndSubInst, ctx.ComponentDescriptor, imports)
	if err != nil {
		return nil, nil, err
	}

	// contains the data objects available for this installation and all of its siblings
	dataObjectsCurrentInstAndSiblings := make(map[string]interface{})
	// contains the targets available for this installation and all of its siblings
	targetsCurrentInstAndSiblings := make(map[string]interface{})

	mergeMaps(dataObjectsCurrentInstAndSiblings, dataImports)
	mergeMaps(targetsCurrentInstAndSiblings, targetImports)

	for _, subInstallation := range renderedDeployItemsAndSubInst.Installations {
		// contains the data object imports for the sub-installation
		// this is a subset of "dataObjectsCurrentInstAndSiblings" containing only the data objects that are explicitly imported by the sub-installation
		subInstDataObjectImports := make(map[string]interface{})
		// contains the target imports for the sub-installation
		// this is a subset of "targetsCurrentInstAndSiblings" containing only the targets that are explicitly imported by the sub-installation
		subInstTargetImports := make(map[string]interface{})

		subInstallationPath := path.Join(pathString, subInstallation.Name)

		// data object imports
		for _, dataImport := range subInstallation.Installation.Spec.Imports.Data {
			v, ok := dataObjectsCurrentInstAndSiblings[dataImport.DataRef]
			if !ok {
				return nil, nil, fmt.Errorf("unable to find data import %s for installation %s", dataImport.DataRef, subInstallationPath)
			}
			subInstDataObjectImports[dataImport.Name] = v
		}

		// target imports
		for _, targetImport := range subInstallation.Installation.Spec.Imports.Targets {
			var (
				ok     bool
				target interface{}
			)

			if len(targetImport.Target) > 0 {
				// single target
				target, ok = targetsCurrentInstAndSiblings[targetImport.Target]
				if !ok {
					return nil, nil, fmt.Errorf("unable to find target import %s for installation %s", targetImport.Target, subInstallationPath)
				}
			} else if len(targetImport.Targets) > 0 {
				// target list
				targetList := make([]interface{}, 0, len(targetImport.Targets))
				for _, t := range targetImport.Targets {
					target, ok = targetsCurrentInstAndSiblings[t]
					if !ok {
						return nil, nil, fmt.Errorf("unable to find target list import %s for installation %s", t, subInstallationPath)
					}
					targetList = append(targetList, target)
				}
				target = targetList
			} else if len(targetImport.TargetListReference) > 0 {
				// target list reference
				target, ok = targetsCurrentInstAndSiblings[targetImport.TargetListReference]
				if !ok {
					return nil, nil, fmt.Errorf("unable to find target list reference import %s for installation %s", targetImport.TargetListReference, subInstallationPath)
				}
			} else {
				return nil, nil, fmt.Errorf("either target, targets or targetListRef must be specified for import %s for installation %s", targetImport.Name, subInstallationPath)
			}

			subInstTargetImports[targetImport.Name] = target
		}

		// execute import data mappings
		err = s.handleDataMappings(subInstallationPath, subInstallation.Spec.ImportDataMappings, subInstDataObjectImports)
		if err != nil {
			return nil, nil, err
		}

		// render the sub-installation
		// subInstExports, _, err := s.executeInstallation(&subInstallation, installationPath, subInstDataObjectImports, subInstTargetImports)
		subInstExports, err := s.handleSubInstallation(installationPath, &subInstallation, subInstDataObjectImports, subInstTargetImports)
		if err != nil {
			return nil, nil, err
		}

		// make the exports available for this installation and its siblings
		mergeMaps(dataObjectsCurrentInstAndSiblings, subInstExports.DataObjects)
		mergeMaps(targetsCurrentInstAndSiblings, subInstExports.Targets)
	}

	// render export executions
	renderedExports, err := s.blueprintRenderer.RenderExportExecutions(ctx, imports, dataObjectsCurrentInstAndSiblings, targetsCurrentInstAndSiblings, exportsByDeployItem)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render exports for installation %s: %w", pathString, err)
	}

	currInstallationExports := InstallationExports{
		DataObjects: make(map[string]interface{}),
		Targets:     make(map[string]interface{}),
	}

	// collect all data objects that are exported by the current installation via export definition
	for _, dataExport := range ctx.Installation.Spec.Exports.Data {
		v, ok := dataObjectsCurrentInstAndSiblings[dataExport.DataRef]
		if ok {
			currInstallationExports.DataObjects[dataExport.Name] = v
		}
		v, ok = renderedExports[dataExport.Name]
		if ok {
			currInstallationExports.DataObjects[dataExport.Name] = v
		}
	}

	// collect all targets that are exported by the current installation via export definition
	for _, targetExport := range ctx.Installation.Spec.Exports.Targets {
		v, ok := targetsCurrentInstAndSiblings[targetExport.Target]
		if ok {
			currInstallationExports.Targets[targetExport.Name] = v
		}
		v, ok = renderedExports[targetExport.Target]
		if ok {
			// the rendered target exports need to converted to a landscaper installation resource
			target, err := convertTargetSpecToTarget(targetExport.Name, "default", v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert target export %s of installation %s to landscaper target type: %w", targetExport.Name, pathString, err)
			}
			currInstallationExports.Targets[targetExport.Name] = target
		}
	}

	// execute export data mappings
	err = s.handleDataMappings(pathString, ctx.Installation.Spec.ExportDataMappings, currInstallationExports.DataObjects)
	if err != nil {
		return nil, nil, err
	}

	currBlueprintExports := BlueprintExports{
		DataObjects: make(map[string]interface{}),
		Targets:     make(map[string]interface{}),
	}

	// collect all blueprint data object and target exports
	for _, export := range ctx.Blueprint.Info.Exports {
		if export.Type == lsv1alpha1.ExportTypeData {
			// data object
			v, ok := dataObjectsCurrentInstAndSiblings[export.Name]
			if ok {
				currBlueprintExports.DataObjects[export.Name] = v
			}
			v, ok = renderedExports[export.Name]
			if ok {
				currBlueprintExports.DataObjects[export.Name] = v
			}
		} else {
			// target
			v, ok := targetsCurrentInstAndSiblings[export.Name]
			if ok {
				currBlueprintExports.Targets[export.Name] = v
			}
			v, ok = renderedExports[export.Name]
			if ok {
				// the rendered target exports need to converted to a landscaper installation resource
				target, err := convertTargetSpecToTarget(export.Name, "default", v)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert target export %s of installation %s to landscaper target type: %w", export.Name, pathString, err)
				}
				currBlueprintExports.Targets[export.Name] = target
			}
		}
	}

	// copy all data object and target exports into a single map which can be used in the exports callback
	dataObjectAndTargetExports := make(map[string]interface{})
	// When the current installation is the "root" installation, there are no exports specified in the installation resource.
	// In that case all exports defined in the root installation blueprint will be exported.
	if _, ok := ctx.Installation.Annotations[rootInstallationAnnotation]; ok {
		mergeMaps(dataObjectAndTargetExports, currBlueprintExports.DataObjects)
		mergeMaps(dataObjectAndTargetExports, currBlueprintExports.Targets)
	} else {
		mergeMaps(dataObjectAndTargetExports, currInstallationExports.DataObjects)
		mergeMaps(dataObjectAndTargetExports, currInstallationExports.Targets)
	}

	s.callbacks.OnExports(pathString, dataObjectAndTargetExports)

	return &currInstallationExports, &currBlueprintExports, nil
}

// handleSubInstallation tries to find an installation export template for the given subinstallation.
// If no matching installation export template was found, the subinstallation is being executed.
func (s *InstallationSimulator) handleSubInstallation(installationPath *InstallationPath, subInstallation *ResolvedInstallation, dataImports, targetImports map[string]interface{}) (*InstallationExports, error) {
	subInstallationPath := installationPath.Child(subInstallation.Installation.Name)
	subInstallationPathString := subInstallationPath.String()

	for _, installationTemplate := range s.exportTemplates.InstallationExports {
		if installationTemplate.SelectorRegexp == nil {
			continue
		}

		if installationTemplate.SelectorRegexp.MatchString(subInstallationPathString) {
			imports := make(map[string]interface{})
			mergeMaps(imports, dataImports)
			mergeMaps(imports, targetImports)

			s.callbacks.OnInstallation(subInstallationPathString, subInstallation.Installation)
			s.callbacks.OnImports(subInstallationPathString, imports)

			templateInput := map[string]interface{}{
				"imports":          imports,
				"installationPath": subInstallationPathString,
			}

			installationEncoded, err := encodeTemplateInput(subInstallation.Installation)
			if err != nil {
				return nil, fmt.Errorf("failed to encode instalation %s: %w", subInstallationPathString, err)
			}
			templateInput["installation"] = installationEncoded

			cdEncoded, err := encodeTemplateInput(subInstallation.ComponentDescriptor)
			if err != nil {
				return nil, fmt.Errorf("failed to encode component descriptor for installation %s: %w", subInstallationPathString, err)
			}
			templateInput["cd"] = cdEncoded

			componentsEncoded, err := encodeTemplateInput(s.blueprintRenderer.cdList)
			if err != nil {
				return nil, fmt.Errorf("failed to encode component descriptor list for installation %s: %w", subInstallationPathString, err)
			}
			templateInput["components"] = componentsEncoded

			out, err := executeTemplate(installationTemplate.Name, installationTemplate.Template, templateInput)
			if err != nil {
				return nil, err
			}

			exports := InstallationExports{}

			dataExports, ok := out["dataExports"]
			if !ok {
				return nil, fmt.Errorf("template output of export template %s has no data export key", installationTemplate.Name)
			}
			exports.DataObjects, ok = dataExports.(map[string]interface{})
			if !ok {
				exports.DataObjects = make(map[string]interface{})
			}

			targetExports, ok := out["targetExports"]
			if !ok {
				return nil, fmt.Errorf("template output of export template %s has no target export key", installationTemplate.Name)
			}
			exports.Targets, ok = targetExports.(map[string]interface{})
			if !ok {
				exports.Targets = make(map[string]interface{})
			}

			dataObjectAndTargetExports := make(map[string]interface{})
			mergeMaps(dataObjectAndTargetExports, exports.DataObjects)
			mergeMaps(dataObjectAndTargetExports, exports.Targets)
			s.callbacks.OnExports(subInstallationPathString, dataObjectAndTargetExports)

			return &exports, nil
		}
	}

	subInstExports, _, err := s.executeInstallation(subInstallation, installationPath, dataImports, targetImports)
	return subInstExports, err
}

// handleDeployItems handles the export calculation of the deploy items defined for an installation.
func (s *InstallationSimulator) handleDeployItems(installationPath string, renderedDeployItems *RenderedDeployItemsSubInstallations, cd *cdv2.ComponentDescriptor, imports map[string]interface{}) (map[string]interface{}, error) {
	exportsByDeployItem := make(map[string]interface{})

	if len(renderedDeployItems.DeployItemTemplateState) > 0 {
		s.callbacks.OnDeployItemTemplateState(installationPath, renderedDeployItems.DeployItemTemplateState)
	}

	for _, deployItem := range renderedDeployItems.DeployItems {
		s.callbacks.OnDeployItem(installationPath, deployItem)

		for _, exportTemplate := range s.exportTemplates.DeployItemExports {
			if exportTemplate.SelectorRegexp == nil {
				continue
			}
			if exportTemplate.SelectorRegexp.MatchString(path.Join(installationPath, deployItem.Name)) {

				templateInput := map[string]interface{}{
					"imports":          imports,
					"installationPath": installationPath,
					"templateName":     exportTemplate.Name,
					"state":            renderedDeployItems.DeployItemTemplateState,
				}

				deployItemEncoded, err := encodeTemplateInput(deployItem)
				if err != nil {
					return nil, fmt.Errorf("failed to encode deploy item %s: %w", deployItem.Name, err)
				}
				templateInput["deployItem"] = deployItemEncoded

				cdEncoded, err := encodeTemplateInput(cd)
				if err != nil {
					return nil, fmt.Errorf("failed to encode component descriptor for deploy item %s: %w", deployItem.Name, err)
				}
				templateInput["cd"] = cdEncoded

				componentsEncoded, err := encodeTemplateInput(s.blueprintRenderer.cdList)
				if err != nil {
					return nil, fmt.Errorf("failed to encode component descriptor list for deploy item %s: %w", deployItem.Name, err)
				}
				templateInput["components"] = componentsEncoded

				out, err := executeTemplate(exportTemplate.Name, exportTemplate.Template, templateInput)
				if err != nil {
					return nil, err
				}

				exports, ok := out["exports"]
				if !ok {
					return nil, fmt.Errorf("template output of export template %s has no export key", exportTemplate.Name)
				}
				exportsMap, ok := exports.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("export key of export template %s is not a map", exportTemplate.Name)
				}
				exportsByDeployItem[deployItem.Name] = exportsMap
			}
		}
	}

	return exportsByDeployItem, nil
}

// handleDataMappings executes a spiff data mapping template, mapping the input values to output values.
func (s *InstallationSimulator) handleDataMappings(installationPath string, dataMappings map[string]lsv1alpha1.AnyJSON, values map[string]interface{}) error {
	input := make(map[string]interface{})
	mergeMaps(input, values)
	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithValues(input)
	if err != nil {
		return fmt.Errorf("unable to init spiff templater for installation %s: %w", installationPath, err)
	}
	for key, dataMapping := range dataMappings {
		tmpl, err := spiffyaml.Unmarshal(key, dataMapping.RawMessage)
		if err != nil {
			return fmt.Errorf("unable to parse data mapping %s for installation %s: %w", key, installationPath, err)
		}
		res, err := spiff.Cascade(tmpl, nil)
		if err != nil {
			return fmt.Errorf("unable to template data mapping %s for installation %s: %w", key, installationPath, err)
		}
		dataBytes, err := spiffyaml.Marshal(res)
		if err != nil {
			return fmt.Errorf("unable to marshal templated data mapping %s for installation %s: %w", key, installationPath, err)
		}
		var data interface{}
		if err := yaml.Unmarshal(dataBytes, &data); err != nil {
			return fmt.Errorf("unable to unmarshal templated data mapping %s for installation %s: %w", key, installationPath, err)
		}
		values[key] = data
	}
	return nil
}

// mergeMaps copies all elements of b into a.
func mergeMaps(a, b map[string]interface{}) {
	for key, val := range b {
		a[key] = val
	}
}

// executeTemplate executes a go template with the given input parameters.
func executeTemplate(templateName, templateSource string, input map[string]interface{}) (map[string]interface{}, error) {
	tmpl, err := gotmpl.New(templateName).Funcs(gotmpl.FuncMap(sprig.FuncMap())).Option("missingkey=zero").Parse(templateSource)
	if err != nil {
		parseError := gotemplate.TemplateErrorBuilder(err).WithSource(&templateSource).Build()
		return nil, parseError
	}

	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, input); err != nil {
		executeError := gotemplate.TemplateErrorBuilder(err).WithSource(&templateSource).
			WithInput(input, template.NewTemplateInputFormatter(true)).
			Build()
		return nil, executeError
	}

	if err := gotemplate.CreateErrorIfContainsNoValue(data.String(), templateName, input, template.NewTemplateInputFormatter(true)); err != nil {
		return nil, err
	}

	var out map[string]interface{}
	if err := yaml.Unmarshal(data.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output of export template %s: %w", templateName, err)
	}

	return out, nil
}

// convertTargetSpecToTarget converts a target spec map into a landscaper target type.
func convertTargetSpecToTarget(name, namespace string, spec interface{}) (map[string]interface{}, error) {
	target := lsv1alpha1.Target{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	raw, err := yaml.Marshal(target)
	if err != nil {
		return nil, err
	}
	var unmarshalled map[string]interface{}
	err = yaml.Unmarshal(raw, &unmarshalled)
	if err != nil {
		return nil, err
	}
	unmarshalled["spec"] = spec
	return unmarshalled, nil
}

func encodeTemplateInput(in interface{}) (map[string]interface{}, error) {
	raw, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	var encoded map[string]interface{}
	err = yaml.Unmarshal(raw, &encoded)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
