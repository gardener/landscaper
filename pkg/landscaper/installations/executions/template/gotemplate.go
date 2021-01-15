// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/gardener/component-spec/bindings-go/utils/selector"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// GoTemplateExecution is the go template implementation for landscaper templating.
type GoTemplateExecution struct {
	blobResolver ctf.BlobResolver
	state        GenericStateHandler
}

// GoTemplateResult describes the result of go templating.
type GoTemplateResult struct {
	State json.RawMessage `json:"state"`
}

// GoTemplate is the GoTemplate executor for a deploy execution.
func (t *GoTemplateExecution) TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint,
	descriptor *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values map[string]interface{}) ([]byte, error) {
	var rawTemplate string
	if len(tmplExec.Template) != 0 {
		if err := json.Unmarshal(tmplExec.Template, &rawTemplate); err != nil {
			return nil, err
		}
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		rawTemplate = string(rawTemplateBytes)
	}
	if len(rawTemplate) == 0 {
		return nil, fmt.Errorf("no template found")
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getDeployExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := template.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs, descriptor, cdList, t.blobResolver)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values["state"] = state
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeDeployExecutionState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	return data.Bytes(), nil
}

func (t *GoTemplateExecution) TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint, exports interface{}) ([]byte, error) {
	var rawTemplate string
	if len(tmplExec.Template) != 0 {
		if err := json.Unmarshal(tmplExec.Template, &rawTemplate); err != nil {
			return nil, err
		}
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		rawTemplate = string(rawTemplateBytes)
	}
	if len(rawTemplate) == 0 {
		return nil, fmt.Errorf("no template found")
	}

	ctx := context.Background()
	defer ctx.Done()
	state, err := t.getExportExecutionState(ctx, tmplExec)
	if err != nil {
		return nil, fmt.Errorf("unable to load state: %w", err)
	}

	tmpl, err := template.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(LandscaperTplFuncMap(blueprint.Fs, nil, nil, t.blobResolver)).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"values": exports,
		"state":  state,
	}
	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, values); err != nil {
		return nil, err
	}
	if err := t.storeExportExecutionState(ctx, tmplExec, data.Bytes()); err != nil {
		return nil, fmt.Errorf("unable to store state: %w", err)
	}
	return data.Bytes(), nil
}

func (t *GoTemplateExecution) getDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	return t.getState(ctx, "deploy", tmplExec)
}

func (t *GoTemplateExecution) storeDeployExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	return t.storeState(ctx, "deploy", tmplExec, data)
}

func (t *GoTemplateExecution) getExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	return t.getState(ctx, "export", tmplExec)
}

func (t *GoTemplateExecution) storeExportExecutionState(ctx context.Context, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	return t.storeState(ctx, "export", tmplExec, data)
}

func (t *GoTemplateExecution) getState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor) (interface{}, error) {
	if t.state == nil {
		return map[string]interface{}{}, nil
	}
	data, err := t.state.Get(ctx, prefix+tmplExec.Name)
	if err != nil {
		if err == StateNotFoundErr {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	var state interface{}
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func (t *GoTemplateExecution) storeState(ctx context.Context, prefix string, tmplExec lsv1alpha1.TemplateExecutor, data []byte) error {
	if t.state == nil {
		return nil
	}
	res := &GoTemplateResult{}
	if err := yaml.Unmarshal(data, res); err != nil {
		return err
	}
	return t.state.Store(ctx, prefix+tmplExec.Name, res.State)
}

// LandscaperSprigFuncMap returns the sanitized spring function map.
func LandscaperSprigFuncMap() template.FuncMap {
	fm := sprig.FuncMap()
	delete(fm, "env")
	delete(fm, "expandenv")
	return template.FuncMap(fm)
}

// LandscaperTplFuncMap contains all additional landscaper functions that are
// available in the executors templates.
func LandscaperTplFuncMap(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, blobResolver ctf.BlobResolver) map[string]interface{} {
	funcs := map[string]interface{}{
		"readFile": readFileFunc(fs),
		"readDir":  readDir(fs),

		"toYaml": toYAML,

		"parseOCIRef":   parseOCIReference,
		"ociRefRepo":    getOCIReferenceRepository,
		"ociRefVersion": getOCIReferenceVersion,
		"resolve":       resolveArtifactFunc(blobResolver),
	}
	funcs["getResource"] = getResourceGoFunc(cd)
	funcs["getResources"] = getResourcesGoFunc(cd)
	funcs["getComponent"] = getComponentGoFunc(cd, cdList)
	return funcs
}

// readFileFunc returns a function that reads a file from a location in a filesystem
func readFileFunc(fs vfs.FileSystem) func(path string) []byte {
	return func(path string) []byte {
		file, err := vfs.ReadFile(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return file
	}
}

// readDir lists all files of directory
func readDir(fs vfs.FileSystem) func(path string) []os.FileInfo {
	return func(path string) []os.FileInfo {
		files, err := vfs.ReadDir(fs, path)
		if err != nil {
			// maybe we should ignore the error and return an empty byte array
			panic(err)
		}
		return files
	}
}

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// parseOCIReference parses a oci reference string into its repository and version.
// e.g. host:5000/myrepo/myimage:1.0.0 -> ["host:5000/myrepo/myimage:1.0.0", "1.0.0"]
func parseOCIReference(ref string) [2]string {
	splitRef := strings.Split(ref, ":")
	if len(splitRef) < 2 {
		panic("invalid reference")
	}

	return [2]string{
		strings.Join(splitRef[:len(splitRef)-1], ":"),
		splitRef[len(splitRef)-1],
	}
}

// getOCIReferenceVersion returns the version of a oci reference
func getOCIReferenceVersion(ref string) string {
	return parseOCIReference(ref)[1]
}

// getOCIReferenceRepository returns the repository of a oci reference
func getOCIReferenceRepository(ref string) string {
	return parseOCIReference(ref)[0]
}

// resolveArtifactFunc returns a function that can resolve artifact defined by a component descriptor access
func resolveArtifactFunc(blobResolver ctf.BlobResolver) func(access map[string]interface{}) []byte {
	return func(access map[string]interface{}) []byte {
		ctx := context.Background()
		defer ctx.Done()
		var data bytes.Buffer
		if _, err := blobResolver.Resolve(ctx, cdv2.Resource{Access: cdv2.NewUnstructuredType(access["type"].(string), access)}, &data); err != nil {
			panic(err)
		}
		return data.Bytes()
	}
}

func getResourcesGoFunc(cd *cdv2.ComponentDescriptor) func(...interface{}) []map[string]interface{} {
	return func(args ...interface{}) []map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a resource as no ComponentDescriptor is defined.")
		}
		resources, err := resolveResources(cd, args)
		if err != nil {
			panic(err)
		}

		data, err := json.Marshal(resources)
		if err != nil {
			panic(err)
		}

		parsedResources := []map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedResources); err != nil {
			panic(err)
		}
		return parsedResources
	}
}

func getResourceGoFunc(cd *cdv2.ComponentDescriptor) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a resource as no ComponentDescriptor is defined.")
		}
		resources, err := resolveResources(cd, args)
		if err != nil {
			panic(err)
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err := json.Marshal(resources[0])
		if err != nil {
			panic(err)
		}

		parsedResource := map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedResource); err != nil {
			panic(err)
		}
		return parsedResource
	}
}

func resolveResources(defaultCD *cdv2.ComponentDescriptor, args []interface{}) ([]cdv2.Resource, error) {
	if len(args) < 3 {
		panic("at least 3 arguments are expected")
	}
	// if the first argument is map we use it as the component descriptor
	// otherwise the default one is used
	desc := defaultCD
	if cdMap, ok := args[0].(map[string]interface{}); ok {
		data, err := json.Marshal(cdMap)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
		}
		desc = &cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, desc); err != nil {
			return nil, err
		}
		// resize the arguments to remove the component descriptor and keep the arguments
		args = args[1:]
	}

	if len(args)%2 != 0 {
		return nil, errors.New("odd number of key value pairs")
	}

	// build the selector from key, value pairs
	sel := selector.DefaultSelector{}
	for i := 0; i < len(args); i = i + 2 {
		key, ok := args[i].(string)
		if !ok {
			panic(fmt.Errorf("expect argument %d to be a string", i))
		}
		value, ok := args[i+1].(string)
		if !ok {
			panic(fmt.Errorf("expect argument %d to be a string", i+1))
		}
		sel[key] = value
	}

	resources, err := desc.GetResourcesBySelector(sel)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func getComponentGoFunc(cd *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a component as no ComponentDescriptor is defined.")
		}
		components, err := resolveComponents(cd, list, args)
		if err != nil {
			panic(err)
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err := json.Marshal(components[0])
		if err != nil {
			panic(err)
		}

		parsedComponent := map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedComponent); err != nil {
			panic(err)
		}
		return parsedComponent
	}
}

func resolveComponents(defaultCD *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList, args []interface{}) ([]cdv2.ComponentDescriptor, error) {
	if len(args) < 2 {
		panic("at least 2 arguments are expected")
	}
	// if the first argument is map we use it as the component descriptor
	// otherwise the default one is used
	desc := defaultCD
	if cdMap, ok := args[0].(map[string]interface{}); ok {
		data, err := json.Marshal(cdMap)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
		}
		desc = &cdv2.ComponentDescriptor{}
		if err := codec.Decode(data, desc); err != nil {
			return nil, err
		}
		// resize the arguments to remove the component descriptor and keep the arguments
		args = args[1:]
	}

	if len(args)%2 != 0 {
		return nil, errors.New("odd number of key value pairs")
	}

	// build the selector from key, value pairs
	sel := selector.DefaultSelector{}
	for i := 0; i < len(args); i = i + 2 {
		key, ok := args[i].(string)
		if !ok {
			panic(fmt.Errorf("expect argument %d to be a string", i))
		}
		value, ok := args[i+1].(string)
		if !ok {
			panic(fmt.Errorf("expect argument %d to be a string", i+1))
		}
		sel[key] = value
	}

	compRefs, err := desc.GetComponentReferences(sel)
	if err != nil {
		return nil, err
	}

	components := make([]cdv2.ComponentDescriptor, len(compRefs))
	for i, compRef := range compRefs {
		cd, err := list.GetComponent(compRef.ComponentName, compRef.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve component %s:%s", compRef.Name, compRef.Version)
		}
		components[i] = cd
	}

	return components, nil
}
