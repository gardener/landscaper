// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	gotmpl "text/template"

	"github.com/Masterminds/sprig/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	imagevector "github.com/gardener/image-vector/pkg"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	lstmpl "github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

// LandscaperSprigFuncMap returns the sanitized spring function map.
func LandscaperSprigFuncMap() gotmpl.FuncMap {
	fm := sprig.FuncMap()
	delete(fm, "env")
	delete(fm, "expandenv")
	return gotmpl.FuncMap(fm)
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

		"getResource":          getResourceGoFunc(cd),
		"getResources":         getResourcesGoFunc(cd),
		"getComponent":         getComponentGoFunc(cd, cdList),
		"getRepositoryContext": getEffectiveRepositoryContextGoFunc,

		"generateImageOverwrite": generateImageVectorGoFunc(cd, cdList),
	}
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
		resources, err := lstmpl.ResolveResources(cd, args)
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
		resources, err := lstmpl.ResolveResources(cd, args)
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

func getEffectiveRepositoryContextGoFunc(arg interface{}) map[string]interface{} {
	if arg == nil {
		panic("Unable to get effective component descriptor as no ComponentDescriptor is defined.")
	}

	cdMap, ok := arg.(map[string]interface{})
	if !ok {
		panic("invalid component descriptor")
	}
	data, err := json.Marshal(cdMap)
	if err != nil {
		panic(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}
	cd := &cdv2.ComponentDescriptor{}
	if err := codec.Decode(data, cd); err != nil {
		panic(fmt.Sprintf("invalid component descriptor: %s", err.Error()))
	}

	data, err = json.Marshal(cd.GetEffectiveRepositoryContext())
	if err != nil {
		panic(fmt.Sprintf("unable to serialize repository context: %s", err.Error()))
	}

	parsedRepoCtx := map[string]interface{}{}
	if err := json.Unmarshal(data, &parsedRepoCtx); err != nil {
		panic(fmt.Sprintf("unable to deserialize repository context: %s", err.Error()))
	}
	return parsedRepoCtx
}

func getComponentGoFunc(cd *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		if cd == nil {
			panic("Unable to search for a component as no ComponentDescriptor is defined.")
		}
		components, err := lstmpl.ResolveComponents(cd, list, args)
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

func generateImageVectorGoFunc(cd *cdv2.ComponentDescriptor, list *cdv2.ComponentDescriptorList) func(args ...interface{}) map[string]interface{} {
	return func(args ...interface{}) map[string]interface{} {
		internalCd := cd
		internalComponents := list

		if len(args) > 2 {
			panic("Too many arguments for generateImageOverwrite.")
		}

		if len(args) >= 1 {
			data, err := json.Marshal(args[0])
			if err != nil {
				panic("Unable to marshal first argument to json.")
			}

			internalCd = &cdv2.ComponentDescriptor{}
			if err = codec.Decode(data, internalCd); err != nil {
				panic("Unable to decode first argument to component descriptor.")
			}
		}

		if len(args) == 2 {
			componentsData, err := json.Marshal(args[1])
			if err != nil {
				panic("Unable to marshal second argument to json.")
			}

			internalComponents = &cdv2.ComponentDescriptorList{}
			if err := codec.Decode(componentsData, internalComponents); err != nil {
				panic("Unable to decode second argument to component descriptor list.")
			}
		}

		if internalCd == nil {
			panic("No component descriptor is defined.")
		}

		if internalComponents == nil {
			panic("No component descriptor list is defined.")
		}

		cdResolver, err := ctf.NewListResolver(list)
		if err != nil {
			panic(fmt.Sprintf("list component resolver could not be build: %s", err.Error()))
		}

		vector, err := imagevector.GenerateImageOverwrite(context.TODO(), cdResolver, internalCd, imagevector.GenerateImageOverwriteOptions{
			Components: internalComponents,
		})
		if err != nil {
			panic(err)
		}

		data, err := json.Marshal(vector)
		if err != nil {
			panic(err)
		}

		parsedImageVector := map[string]interface{}{}
		if err := json.Unmarshal(data, &parsedImageVector); err != nil {
			panic(err)
		}
		return parsedImageVector
	}
}
