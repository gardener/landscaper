// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/core/validation"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/execution"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	"github.com/gardener/landscaper/pkg/logger"
)

const (
	OutputResourceDeployItems      = "deployitems"
	OutputResourceSubinstallations = "subinstallations"
)

var (
	OutputResourceAllTerms              = sets.NewString("all")
	OutputResourceDeployItemsTerms      = sets.NewString(OutputResourceDeployItems, "di")
	OutputResourceSubinstallationsTerms = sets.NewString(OutputResourceSubinstallations, "subinst", "inst")
)

const (
	JSONOut = "json"
	YAMLOut = "yaml"
)

type renderOptions struct {
	// blueprintPath is the path to the directory containing the definition.
	blueprintPath string
	// componentDescriptorPath is the path tot he component descriptor to be used
	componentDescriptorPath string
	// componentDescriptorPath is the path tot he component descriptor to be used
	additionalComponentDescriptorPath []string
	// valueFiles is a list of filepaths to value yaml files.
	valueFiles []string
	// outputFormat defines the format of the output
	outputFormat string
	// outDir is the directory where the rendered should be written to.
	outDir string

	outputResources         sets.String
	blueprint               *lsv1alpha1.Blueprint
	blueprintFs             vfs.FileSystem
	componentDescriptor     *cdv2.ComponentDescriptor
	componentDescriptorList *cdv2.ComponentDescriptorList
	values                  *Values
}

// NewRenderCommand creates a new local command to render a blueprint instance locally
func NewRenderCommand(ctx context.Context) *cobra.Command {
	opts := &renderOptions{}
	cmd := &cobra.Command{
		Use:     "render",
		Args:    cobra.RangeArgs(1, 2),
		Example: "landscapercli blueprints render [path to Blueprint directory] [all,deployitems,subinstallations]",
		Short:   "renders the given blueprint",
		Long: fmt.Sprintf(`
Renders the blueprint with the given values files.
All value files are merged whereas the later defined will overwrite the values of the previous ones

By default all rendered resources are printed to stdout.
Specific resources can be printed by adding a second argument.
landscapercli local render [path to Blueprint directory] [resource]
Available resources are
- %s: renders all available resources
- %s: renders deployitems
- %s: renders subinstallations
`,
			strings.Join(OutputResourceAllTerms.List(), "|"),
			strings.Join(OutputResourceDeployItemsTerms.List(), "|"),
			strings.Join(OutputResourceSubinstallationsTerms.List(), "|")),
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.run(ctx, logger.Log); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *renderOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.componentDescriptorPath, "component-descriptor", "c", "", "Path to the local component descriptor")
	fs.StringArrayVarP(&o.additionalComponentDescriptorPath, "additional-component-descriptor", "a", []string{}, "Path to additional local component descriptors")
	fs.StringArrayVarP(&o.valueFiles, "file", "f", []string{}, "List of filepaths to value yaml files that define the imports")
	fs.StringVarP(&o.outputFormat, "output", "o", YAMLOut, "The format of the output. Can be json or yaml.")
	fs.StringVarP(&o.outDir, "write", "w", "", "The output directory where the rendered files should be written to")
}

func (o *renderOptions) run(_ context.Context, log logr.Logger) error {
	log.V(3).Info(fmt.Sprintf("rendering %s", strings.Join(o.outputResources.List(), ", ")))

	blueprint, err := blueprints.New(o.blueprint, o.blueprintFs)
	if err != nil {
		return err
	}

	if o.outputResources.Has(OutputResourceDeployItems) {
		templateStateHandler := template.NewMemoryStateHandler()
		deployItemTemplates, err := template.New(nil, templateStateHandler).
			TemplateDeployExecutions(blueprint, o.componentDescriptor, &cdv2.ComponentDescriptorList{}, o.values.Imports)
		if err != nil {
			return fmt.Errorf("unable to template deploy executions: %w", err)
		}

		// print out state
		stateOut := map[string]map[string]json.RawMessage{
			"state": map[string]json.RawMessage{},
		}
		for key, state := range templateStateHandler {
			stateOut["state"][key] = json.RawMessage(state)
		}
		if err := o.out(stateOut, "state"); err != nil {
			return err
		}

		for _, diTmpl := range deployItemTemplates {
			di := &lsv1alpha1.DeployItem{
				TypeMeta: metav1.TypeMeta{
					APIVersion: lsv1alpha1.SchemeGroupVersion.String(),
					Kind:       "DeployItem",
				},
			}
			execution.ApplyDeployItemTemplate(di, diTmpl)
			if err := o.out(di, "deployitems", diTmpl.Name); err != nil {
				return err
			}
		}
	}

	if o.outputResources.Has(OutputResourceSubinstallations) {
		dummyInst := &lsv1alpha1.Installation{
			TypeMeta: metav1.TypeMeta{
				APIVersion: lsv1alpha1.SchemeGroupVersion.String(),
				Kind:       "Installation",
			},
		}
		if len(blueprint.Subinstallations) == 0 {
			fmt.Printf("No subinstallations defined\n")
		}
		for _, subInstTmpl := range blueprint.Subinstallations {
			subInst := &lsv1alpha1.Installation{}
			subInst.Spec = lsv1alpha1.InstallationSpec{
				Imports:            subInstTmpl.Imports,
				ImportDataMappings: subInstTmpl.ImportDataMappings,
				Exports:            subInstTmpl.Exports,
				ExportDataMappings: subInstTmpl.ExportDataMappings,
			}
			subBlueprint, err := subinstallations.GetBlueprintDefinitionFromInstallationTemplate(
				dummyInst,
				subInstTmpl,
				o.componentDescriptor,
				cdutils.ComponentReferenceResolverFromList(o.componentDescriptorList))
			if err != nil {
				fmt.Printf("unable to get blueprint: %s\n", err.Error())
			} else if subBlueprint != nil {
				subInst.Spec.Blueprint = *subBlueprint
			}
			if err := o.out(subInst, "subinstallations", subInstTmpl.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *renderOptions) Complete(args []string) error {
	o.blueprintPath = args[0]
	data, err := ioutil.ReadFile(filepath.Join(o.blueprintPath, lsv1alpha1.BlueprintFileName))
	if err != nil {
		return fmt.Errorf("unable to read blueprint from %s: %w", filepath.Join(o.blueprintPath, lsv1alpha1.BlueprintFileName), err)
	}
	o.blueprint = &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, o.blueprint); err != nil {
		return err
	}
	o.blueprintFs, err = projectionfs.New(osfs.New(), o.blueprintPath)
	if err != nil {
		return fmt.Errorf("unable to construct blueprint filesystem: %w", err)
	}

	if len(o.componentDescriptorPath) != 0 {
		data, err := ioutil.ReadFile(o.componentDescriptorPath)
		if err != nil {
			return fmt.Errorf("unable to read component descriptor from %s: %w", o.componentDescriptorPath, err)
		}
		cd := &cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, cd); err != nil {
			return fmt.Errorf("unable to decode component descriptor: %w", err)
		}
		o.componentDescriptor = cd
	}

	o.componentDescriptorList = &cdv2.ComponentDescriptorList{}
	for _, cdPath := range o.additionalComponentDescriptorPath {
		data, err := ioutil.ReadFile(cdPath)
		if err != nil {
			return fmt.Errorf("unable to read component descriptor from %s: %w", o.componentDescriptorPath, err)
		}
		cd := cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, &cd); err != nil {
			return fmt.Errorf("unable to decode component descriptor: %w", err)
		}
		o.componentDescriptorList.Components = append(o.componentDescriptorList.Components, cd)
	}

	o.values = &Values{}
	for _, filePath := range o.valueFiles {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("unable to read values file '%s': %w", filePath, err)
		}
		values := &Values{}
		if err := yaml.Unmarshal(data, values); err != nil {
			return fmt.Errorf("unable to parse values file '%s': %w", filePath, err)
		}

		MergeValues(o.values, values)
	}

	if err := o.parseOutputResources(args); err != nil {
		return err
	}

	return o.Validate()
}

// Validate validates push options
func (o *renderOptions) Validate() error {
	blueprint := &core.Blueprint{}
	if err := lsv1alpha1.Convert_v1alpha1_Blueprint_To_core_Blueprint(o.blueprint, blueprint, nil); err != nil {
		return err
	}
	if errList := validation.ValidateBlueprint(o.blueprintFs, blueprint); len(errList) != 0 {
		return errList.ToAggregate()
	}

	if err := ValidateValues(o.values); err != nil {
		return err
	}

	if o.outputFormat != YAMLOut && o.outputFormat != JSONOut {
		return fmt.Errorf("output format is expected to be json or yaml but got '%s'", o.outputFormat)
	}

	return nil
}

func (o *renderOptions) parseOutputResources(args []string) error {
	allResources := sets.NewString(OutputResourceDeployItems, OutputResourceSubinstallations)
	if len(args) == 1 {
		o.outputResources = allResources
		return nil
	}
	if len(args) > 1 {
		resources := strings.Split(args[1], ",")
		o.outputResources = sets.NewString()
		for _, res := range resources {
			if OutputResourceAllTerms.Has(res) {
				o.outputResources = allResources
				return nil
			} else if OutputResourceDeployItemsTerms.Has(res) {
				o.outputResources.Insert(OutputResourceDeployItems)
			} else if OutputResourceSubinstallationsTerms.Has(res) {
				o.outputResources.Insert(OutputResourceSubinstallations)
			} else {
				return fmt.Errorf("unknown resource '%s'", res)
			}
		}
	}
	return nil
}

func (o *renderOptions) out(obj interface{}, names ...string) error {

	var data []byte
	switch o.outputFormat {
	case YAMLOut:
		var err error
		data, err = yaml.Marshal(obj)
		if err != nil {
			return err
		}
	case JSONOut:
		var err error
		data, err = json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output format '%s'", o.outputFormat)
	}

	// print to stdout if no directory is given
	if len(o.outDir) == 0 {

		if len(names) != 0 {
			fmt.Println("--------------------------------------")
			fmt.Printf("-- %s\n", strings.Join(names, " "))
			fmt.Println("--------------------------------------")
		}
		fmt.Printf("%s\n", data)
		return nil
	}

	objFilePath := filepath.Join(append([]string{o.outDir}, names...)...)
	if err := os.MkdirAll(filepath.Dir(objFilePath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create path %s", o.outDir)
	}
	return ioutil.WriteFile(objFilePath, data, os.ModePerm)
}
