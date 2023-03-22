// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// Templater implements all available template executors.
type Templater struct {
	impl map[lsv1alpha1.TemplateType]ExecutionTemplater
}

// New creates a new instance of a templater.
func New(templaters ...ExecutionTemplater) *Templater {
	t := &Templater{
		impl: make(map[lsv1alpha1.TemplateType]ExecutionTemplater),
	}
	for _, templater := range templaters {
		t.impl[templater.Type()] = templater
	}
	return t
}

// ExecutionTemplater describes a implementation for a template execution
type ExecutionTemplater interface {
	// Type returns the type of the templater.
	Type() lsv1alpha1.TemplateType
	// TemplateImportExecutions templates an import executor and return a list of error messages.
	TemplateImportExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*ImportExecutorOutput, error)
	// TemplateSubinstallationExecutions templates a subinstallation executor and return a list of installations templates.
	TemplateSubinstallationExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*SubinstallationExecutorOutput, error)
	// TemplateDeployExecutions templates a deploy executor and return a list of deployitem templates.
	TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*DeployExecutorOutput, error)
	// TemplateExportExecutions templates a export executor.
	// It return the exported data as key value map where the key is the name of the export.
	TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		descriptor *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*ExportExecutorOutput, error)
}

// SubinstallationExecutorOutput describes the output of deploy executor.
type SubinstallationExecutorOutput struct {
	Subinstallations []*lsv1alpha1.InstallationTemplate `json:"subinstallations"`
}

func (o SubinstallationExecutorOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(o)
}

func (o *SubinstallationExecutorOutput) UnmarshalJSON(data []byte) error {
	type helperStruct struct {
		Subinstallations []json.RawMessage `json:"subinstallations"`
	}
	rawList := &helperStruct{}
	if err := json.Unmarshal(data, rawList); err != nil {
		return err
	}

	out := SubinstallationExecutorOutput{
		Subinstallations: make([]*lsv1alpha1.InstallationTemplate, len(rawList.Subinstallations)),
	}
	for i, raw := range rawList.Subinstallations {
		instTmpl := lsv1alpha1.InstallationTemplate{}
		if _, _, err := api.Decoder.Decode(raw, nil, &instTmpl); err != nil {
			return fmt.Errorf("unable to decode installation template %d: %w", i, err)
		}
		out.Subinstallations[i] = &instTmpl
	}

	*o = out
	return nil
}

// ImportExecutorOutput describes the output of import executor.
type ImportExecutorOutput struct {
	Bindings map[string]interface{} `json:"bindings"`
	Errors   []string               `json:"errors"`
}

type TargetReference struct {
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Import string `json:"import,omitempty"`

	// +optional
	Index *int `json:"index,omitempty"`
}

// DeployItemSpecification defines a execution element that is translated into a deployitem template for the execution object.
type DeployItemSpecification struct {
	// Name is the unique name of the execution.
	Name string `json:"name"`

	// DataType is the DeployItem type of the execution.
	Type core.DeployItemType `json:"type"`

	// Target is the target reference to the target import of the target the deploy item should deploy to.
	// +optional
	Target *TargetReference `json:"target,omitempty"`

	// Labels is the map of labels to be added to the deploy item.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ProviderConfiguration contains the type specific configuration for the execution.
	Configuration *runtime.RawExtension `json:"config"`

	// DependsOn lists deploy items that need to be executed before this one
	DependsOn []string `json:"dependsOn,omitempty"`

	// Timeout specifies how long the deployer may take to apply the deploy item.
	// When the time is exceeded, the deploy item fails.
	// Value has to be parsable by time.ParseDuration (or 'none' to deactivate the timeout).
	// Defaults to ten minutes if not specified.
	// +optional
	Timeout *lsv1alpha1.Duration `json:"timeout,omitempty"`

	// UpdateOnChangeOnly specifies if redeployment is executed only if the specification of the deploy item has changed.
	// +optional
	UpdateOnChangeOnly bool `json:"updateOnChangeOnly,omitempty"`
}

// DeployExecutorOutput describes the output of deploy executor.
type DeployExecutorOutput struct {
	DeployItems []DeployItemSpecification `json:"deployItems"`
}

// ExportExecutorOutput describes the output of export executor.
type ExportExecutorOutput struct {
	Exports map[string]interface{} `json:"exports"`
}

func (o *Templater) TemplateImportExecutions(opts BlueprintExecutionOptions) ([]string, map[string]interface{}, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, nil, err
	}

	errorList := []string{}
	bindings := map[string]interface{}{}

	for _, tmplExec := range opts.Blueprint.Info.ImportExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateImportExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, nil, err
		}
		if output.Bindings != nil {
			var imports map[string]interface{}
			imp := values["imports"]
			if imports == nil {
				imports = map[string]interface{}{}
				values["imports"] = imports
			} else {
				imports = imp.(map[string]interface{})
			}
			for k, v := range output.Bindings {
				bindings[k] = v
				imports[k] = v
			}
		}
		if len(output.Errors) != 0 {
			errorList = append(errorList, output.Errors...)
			break
		}
	}

	return errorList, bindings, nil
}

// TemplateSubinstallationExecutions templates all subinstallation executions and
// returns a aggregated list of all templated installation templates.
func (o *Templater) TemplateSubinstallationExecutions(opts DeployExecutionOptions) ([]*lsv1alpha1.InstallationTemplate, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, err
	}
	installationTemplates := make([]*lsv1alpha1.InstallationTemplate, 0)
	for _, tmplExec := range opts.Blueprint.Info.SubinstallationExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateSubinstallationExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		if output.Subinstallations == nil {
			continue
		}
		installationTemplates = append(installationTemplates, output.Subinstallations...)
	}

	return installationTemplates, nil
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateDeployExecutions(opts DeployExecutionOptions) ([]DeployItemSpecification, error) {

	values, err := opts.Values()
	if err != nil {
		return nil, err
	}

	deployItemTemplateList := []DeployItemSpecification{}
	for _, tmplExec := range opts.Blueprint.Info.DeployExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateDeployExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		if output.DeployItems == nil {
			continue
		}
		deployItemTemplateList = append(deployItemTemplateList, output.DeployItems...)
	}

	return deployItemTemplateList, nil
}

// TemplateExportExecutions templates all exports.
func (o *Templater) TemplateExportExecutions(opts ExportExecutionOptions) (map[string]interface{}, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, err
	}
	exportData := make(map[string]interface{})
	for _, tmplExec := range opts.Blueprint.Info.ExportExecutions {

		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateExportExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		exportData = utils.MergeMaps(exportData, output.Exports)
	}

	return exportData, nil
}

func serializeComponentDescriptor(cd *cdv2.ComponentDescriptor) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

func serializeComponentDescriptorList(cd *cdv2.ComponentDescriptorList) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}
