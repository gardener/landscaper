// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package spiff

import (
	"context"
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	imagevector "github.com/gardener/image-vector/pkg"
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

func LandscaperSpiffFuncs(functions spiffing.Functions, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) {
	functions.RegisterFunction("getResource", spiffResolveResources(cd))
	functions.RegisterFunction("getComponent", spiffResolveComponent(cd, cdList))
	functions.RegisterFunction("generateImageOverwrite", spiffGenerateImageOverwrite(cd, cdList))
	functions.RegisterFunction("parseOCIRef", parseOCIReference)
	functions.RegisterFunction("ociRefRepo", getOCIReferenceRepository)
	functions.RegisterFunction("ociRefVersion", getOCIReferenceVersion)
}

func spiffResolveResources(cd *cdv2.ComponentDescriptor) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		resources, err := template.ResolveResources(cd, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(resources[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func spiffResolveComponent(cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()
		data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments, ""))
		if err != nil {
			return info.Error(err.Error())
		}
		var val []interface{}
		if err := yaml.Unmarshal(data, &val); err != nil {
			return info.Error(err.Error())
		}

		components, err := template.ResolveComponents(cd, cdList, val)
		if err != nil {
			return info.Error(err.Error())
		}

		// resources must be at least one, otherwise an error will be thrown
		data, err = json.Marshal(components[0])
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}
		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func spiffGenerateImageOverwrite(cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	return func(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
		info := dynaml.DefaultInfo()

		internalCd := cd
		internalComponents := cdList

		if len(arguments) > 2 {
			return info.Error("Too many arguments for generateImageOverwrite.")
		}

		if len(arguments) >= 1 {
			data, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments[0], ""))
			if err != nil {
				return info.Error(err.Error())
			}

			internalCd = &cdv2.ComponentDescriptor{}
			if err := yaml.Unmarshal(data, internalCd); err != nil {
				return info.Error(err.Error())
			}
		}

		if len(arguments) == 2 {
			componentsData, err := spiffyaml.Marshal(spiffyaml.NewNode(arguments[1], ""))
			if err != nil {
				return info.Error(err.Error())
			}

			internalComponents = &cdv2.ComponentDescriptorList{}
			if err := yaml.Unmarshal(componentsData, internalComponents); err != nil {
				return info.Error(err.Error())
			}
		}

		if internalCd == nil {
			return info.Error("No component descriptor is defined.")
		}

		if internalComponents == nil {
			return info.Error("No component descriptor list is defined.")
		}

		cdResolver, err := ctf.NewListResolver(cdList)
		if err != nil {
			return info.Error("list component resolver could not be build: %s", err.Error())
		}

		vector, err := imagevector.GenerateImageOverwrite(context.TODO(), cdResolver, internalCd, imagevector.GenerateImageOverwriteOptions{
			Components: internalComponents,
		})
		if err != nil {
			return info.Error(err.Error())
		}

		data, err := yaml.Marshal(vector)
		if err != nil {
			return info.Error(err.Error())
		}

		node, err := spiffyaml.Parse("", data)
		if err != nil {
			return info.Error(err.Error())
		}

		result, err := binding.Flow(node, false)
		if err != nil {
			return info.Error(err.Error())
		}

		return result.Value(), info, true
	}
}

func parseOCIReference(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref, ok := arguments[0].(string)
	if !ok {
		return info.Error("Invalid argument: string expected")
	}
	data, err := yaml.Marshal(template.ParseOCIReference(ref))
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}

func getOCIReferenceRepository(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref := arguments[0].(string)
	data, err := yaml.Marshal(template.ParseOCIReference(ref)[0])
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}

func getOCIReferenceVersion(arguments []interface{}, binding dynaml.Binding) (interface{}, dynaml.EvaluationInfo, bool) {
	info := dynaml.DefaultInfo()
	if len(arguments) > 1 {
		return info.Error("Too many arguments for parseOCIReference. Expected 1 reference.")
	}
	ref := arguments[0].(string)
	data, err := yaml.Marshal(template.ParseOCIReference(ref)[1])
	if err != nil {
		return info.Error(err.Error())
	}

	node, err := spiffyaml.Parse("", data)
	if err != nil {
		return info.Error(err.Error())
	}

	result, err := binding.Flow(node, false)
	if err != nil {
		return info.Error(err.Error())
	}

	return result.Value(), info, true
}
