// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gardener/component-cli/ociclient/oci"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/utils/selector"
)

// ResolveResources is a helper function that can be used in the specific templating implementations to
// ease the access to component descriptor defined resources.
// The method takes a default component descriptor and a optional number of args.
//
// The arguments are expected to be a set of key value pairs that describe the identity of the resource.
// e.g. []interface{}{"name", "my-resource"}.
// Optionally the first argument can be a component descriptor provided as map[string]interface{}
//
func ResolveResources(defaultCD *cdv2.ComponentDescriptor, args []interface{}) ([]cdv2.Resource, error) {
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

	resources, err := desc.GetResourcesBySelector(sel)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

// ResolveComponents is a helper function that can be used in the specific templating implementations to
// ease the access to component descriptors.
// The method takes a default component descriptor,  a list of components and a optional number of args.
//
// The arguments are expected to be a set of key value pairs that describe the identity of the resource.
// e.g. []interface{}{"name", "my-component"}.
// Optionally the first argument can be a component descriptor provided as map[string]interface{}
//
func ResolveComponents(defaultCD *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList, args []interface{}) ([]cdv2.ComponentDescriptor, error) {
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

// ParseOCIReference parses a oci reference string into its repository and version.
// e.g. host:5000/myrepo/myimage:1.0.0 -> ["host:5000/myrepo/myimage", "1.0.0"]
// host:5000/myrepo/myimage@sha256:123 -> ["host:5000/myrepo/myimage", "sha256:123"]
func ParseOCIReference(ref string) [2]string {
	refspec, err := oci.ParseRef(ref)
	if err != nil {
		panic(err)
	}
	splitRef := strings.Split(ref, ":")
	if len(splitRef) < 2 {
		panic("invalid reference")
	}

	// todo: remove workaround with new component-cli version
	repository := strings.TrimPrefix(refspec.Name(), "index.docker.io/library/")

	if refspec.Tag != nil {
		return [2]string{
			repository,
			*refspec.Tag,
		}
	} else if refspec.Digest != nil {
		return [2]string{
			repository,
			refspec.Digest.String(),
		}
	}
	return [2]string{
		repository,
		"",
	}
}
