package landscaper

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"
	"regexp"
	"sigs.k8s.io/yaml"
	gotmpl "text/template"
)

const (
	rootInstallationName = "root"
)

type InstallationPath struct {
	name string
	parent *InstallationPath
}

func NewInstallationPath(name string) *InstallationPath {
	return &InstallationPath{
		name:   name,
		parent: nil,
	}
}

func (p *InstallationPath) Child(name string) *InstallationPath {
	return &InstallationPath{
		name:   name,
		parent: p,
	}
}

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

type ExportTemplates struct {
	DeployItemExports []*DeployItemExportTemplate `json:"deployItems"`
}

type DeployItemExportTemplate struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
	Template string `json:"template"`

	SelectorRegexp *regexp.Regexp `json:"-"`
}

type InstallationSimulatorCallbacks interface {
	OnInstallation(path string, installation *lsv1alpha1.Installation)
	OnImports(path string, imports map[string]interface{})
	OnDeployItem(path string, deployItem *lsv1alpha1.DeployItem)
	OnExports(path string, exports map[string]interface{})
}

type emptySimulatorCallbacks struct {
}

func (c emptySimulatorCallbacks) OnInstallation(_ string, _ *lsv1alpha1.Installation) {}
func (c emptySimulatorCallbacks) OnImports(_ string, _ map[string]interface{})        {}
func (c emptySimulatorCallbacks) OnDeployItem(_ string, _ *lsv1alpha1.DeployItem)     {}
func (c emptySimulatorCallbacks) OnExports(_ string, _ map[string]interface{})        {}

type Exports struct {
	DataObjects map[string]interface{}
	Targets map[string]interface{}
}

type InstallationSimulator struct {
	blueprintRenderer *BlueprintRenderer
	exportTemplates   ExportTemplates
	callbacks         InstallationSimulatorCallbacks
}

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

	return &InstallationSimulator{
		blueprintRenderer: NewBlueprintRenderer(cdList, resolver, repositoryContext),
		exportTemplates:   exportTemplates,
		callbacks:         emptySimulatorCallbacks{},
	}, nil
}

func (s *InstallationSimulator) SetCallbacks(callbacks InstallationSimulatorCallbacks) *InstallationSimulator {
	s.callbacks = callbacks
	return s
}

func (s *InstallationSimulator) Run(cd *cdv2.ComponentDescriptor, blueprint *blueprints.Blueprint, imports map[string]interface{}) (*Exports, error) {
	ctx := &RenderInput{
		ComponentDescriptor: cd,
		Installation: &lsv1alpha1.Installation{
			ObjectMeta: metav1.ObjectMeta{
				Name: rootInstallationName,
			},
		},
		Blueprint: blueprint,
	}

	return s.executeInstallation(ctx, nil, imports, imports)
}

func (s *InstallationSimulator) executeInstallation(ctx *RenderInput, installationPath *InstallationPath, dataImports, targetImports map[string]interface{}) (*Exports, error) {
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
		return nil, fmt.Errorf("failed to render deploy items and subinstallations %q: %w", pathString, err)
	}

	exportsByDeployItem, err := s.handleDeployItems(pathString, renderedDeployItemsAndSubInst, imports)
	if err != nil {
		return nil, err
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

		for _, dataImport := range subInstallation.Installation.Spec.Imports.Data {
			v, ok := dataObjectsCurrentInstAndSiblings[dataImport.DataRef]
			if !ok {
				return nil, fmt.Errorf("unable to find data import %s for installation %s", dataImport.DataRef, subInstallationPath)
			}
			subInstDataObjectImports[dataImport.Name] = v
		}

		for _, targetImport := range subInstallation.Installation.Spec.Imports.Targets {
			v, ok := targetsCurrentInstAndSiblings[targetImport.Target]
			if !ok {
				return nil, fmt.Errorf("unable to find target import %s for installation %s", targetImport.Target, subInstallationPath)
			}
			subInstTargetImports[targetImport.Name] = v
		}

		// execute import data mappings
		err = s.handleDataMappings(subInstallationPath, subInstallation.Spec.ImportDataMappings, subInstDataObjectImports, subInstDataObjectImports)
		if err != nil {
			return nil, err
		}

		// render the sub-installation
		subInstExports, err := s.executeInstallation((*RenderInput)(&subInstallation), installationPath, subInstDataObjectImports, subInstTargetImports)
		if err != nil {
			return nil, err
		}

		// make the exports available for this installation and its siblings
		mergeMaps(dataObjectsCurrentInstAndSiblings, subInstExports.DataObjects)
		mergeMaps(targetsCurrentInstAndSiblings, subInstExports.Targets)
	}

	// render export executions
	renderedExports, err := s.blueprintRenderer.RenderExportExecutions(ctx, dataObjectsCurrentInstAndSiblings, targetsCurrentInstAndSiblings, exportsByDeployItem)
	if err != nil {
		return nil, fmt.Errorf("failed to render exports for installation %s: %w", pathString, err)
	}

	currInstallationExports := Exports{
		DataObjects: make(map[string]interface{}),
		Targets:     make(map[string]interface{}),
	}

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

	for _, targetExport := range ctx.Installation.Spec.Exports.Targets {
		v, ok := targetsCurrentInstAndSiblings[targetExport.Target]
		if ok {
			currInstallationExports.Targets[targetExport.Name] = v
		}
		v, ok = renderedExports[targetExport.Target]
		if ok {
			target, err := convertTargetSpecToTarget(targetExport.Name, "default", v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert target export %s of installation %s to landscaper target type: %w", targetExport.Name, pathString, err)
			}
			currInstallationExports.Targets[targetExport.Name] = target
		}
	}

	// execute export data mappings
	err = s.handleDataMappings(pathString, ctx.Installation.Spec.ExportDataMappings, dataObjectsCurrentInstAndSiblings, currInstallationExports.DataObjects)
	if err != nil {
		return nil, err
	}

	dataObjectAndTargetExports := make(map[string]interface{})
	mergeMaps(dataObjectAndTargetExports, currInstallationExports.DataObjects)
	mergeMaps(dataObjectAndTargetExports, currInstallationExports.Targets)

	if pathString == rootInstallationName {
		for _, export := range ctx.Blueprint.Info.Exports {
			v, ok := renderedExports[export.Name]
			if ok {
				dataObjectAndTargetExports[export.Name] = v
			}
			v, ok = dataObjectsCurrentInstAndSiblings[export.Name]
			if ok {
				dataObjectAndTargetExports[export.Name] = v
			}
			v, ok = targetsCurrentInstAndSiblings[export.Name]
			if ok {
				dataObjectAndTargetExports[export.Name] = v
			}
		}
	}

	s.callbacks.OnExports(pathString, dataObjectAndTargetExports)

	return &currInstallationExports, nil
}

func (s *InstallationSimulator) handleDeployItems(installationPath string, renderedDeployItems *RenderedDeployItemsSubInstallations, imports map[string]interface{}) (map[string]interface{}, error) {
	exportsByDeployItem := make(map[string]interface{})

	for _, deployItem := range renderedDeployItems.DeployItems {
		s.callbacks.OnDeployItem(installationPath, deployItem)

		for _, exportTemplate := range s.exportTemplates.DeployItemExports {
			if exportTemplate.SelectorRegexp == nil {
				continue
			}
			if exportTemplate.SelectorRegexp.MatchString(path.Join(installationPath, deployItem.Name)) {
				templateInput := map[string]interface{} {
					"imports": imports,
					"installationPath": installationPath,
					"templateName": exportTemplate.Name,
				}

				tmpl, err := gotmpl.New(exportTemplate.Name).Funcs(gotmpl.FuncMap(sprig.FuncMap())).Option("missingkey=zero").Parse(exportTemplate.Template)
				if err != nil {
					parseError := gotemplate.TemplateErrorBuilder(err).WithSource(&exportTemplate.Template).Build()
					return nil, parseError
				}

				data := bytes.NewBuffer([]byte{})
				if err := tmpl.Execute(data, imports); err != nil {
					executeError := gotemplate.TemplateErrorBuilder(err).WithSource(&exportTemplate.Template).
						WithInput(templateInput, template.NewTemplateInputFormatter(true)).
						Build()
					return nil, executeError
				}

				if err := gotemplate.CreateErrorIfContainsNoValue(data.String(), exportTemplate.Name, templateInput, template.NewTemplateInputFormatter(true)); err != nil {
					return nil, err
				}

				var out map[string]interface{}
				if err := yaml.Unmarshal(data.Bytes(), &out); err != nil {
					return nil, fmt.Errorf("failed to unmarshal output of export template %s: %w", exportTemplate.Name, err)
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

func (s *InstallationSimulator) handleDataMappings(installationPath string, dataMappings map[string]lsv1alpha1.AnyJSON, input, output map[string]interface{}) error {
	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithValues(input)
	if err != nil {
		return fmt.Errorf("unable to init spiff templater for installation %s: %w", installationPath, err)
	}
	for key, dataMapping := range dataMappings {
		tmpl, err := spiffyaml.Unmarshal(key, dataMapping.RawMessage)
		if err != nil {
			return fmt.Errorf("unable to parse import mapping %s for installation %s: %w", key, installationPath, err)
		}
		res, err := spiff.Cascade(tmpl, nil)
		if err != nil {
			return fmt.Errorf("unable to template import mapping %s for installation %s: %w", key, installationPath, err)
		}
		dataBytes, err := spiffyaml.Marshal(res)
		if err != nil {
			return fmt.Errorf("unable to marshal templated import mapping %s for installation %s: %w", key, installationPath, err)
		}
		var data interface{}
		if err := yaml.Unmarshal(dataBytes, &data); err != nil {
			return fmt.Errorf("unable to unmarshal templated import mapping %s for installation %s: %w", key, installationPath, err)
		}
		output[key] = data
	}
	return nil
}

func mergeMaps(a, b map[string]interface{}) {
	for key, val := range b {
		a[key] = val
	}
}

func convertTargetSpecToTarget(name, namespace string, spec interface{}) (map[string]interface{}, error) {
	target := lsv1alpha1.Target{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
