// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gardener/landscaper/pkg/components/model/types"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// ReferenceContext describes the context of the current reference.
type ReferenceContext struct {
	// LocalTypes is a map of blueprint locally defined types.
	// It is a map of schema name to schema definition
	LocalTypes map[string]lsv1alpha1.JSONSchemaDefinition
	// BlueprintFs is the virtual filesystem that is used to resolve "blueprint" refs
	BlueprintFs vfs.FileSystem
	// ComponentVersion contains the current blueprint's component descriptor.
	ComponentVersion model.ComponentVersion
	// RegistryAccess is an object that can resolve component descriptors.
	RegistryAccess model.RegistryAccess
	// RepositoryContext can be used to overwrite the effective repository context of the component descriptor.
	// If not set, the effective repository context of the ComponentDescriptor will be used.
	RepositoryContext *types.UnstructuredTypedObject
}

type ReferenceResolver struct {
	*ReferenceContext
}

func NewReferenceResolver(refCtx *ReferenceContext) *ReferenceResolver {
	if refCtx == nil {
		refCtx = &ReferenceContext{}
	}
	return &ReferenceResolver{refCtx}
}

// Resolve walks through the given json schema and recursively resolves all references which use one of
// the "local", "blueprint", or "cd" schemes.
func (rr *ReferenceResolver) Resolve(schemaBytes []byte) (interface{}, error) {
	data, err := decodeJSON(schemaBytes)
	if err != nil {
		return nil, err
	}
	return rr.resolve(data, field.NewPath(""), nil)
}

func (rr *ReferenceResolver) resolve(data interface{}, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	switch typedData := data.(type) {
	case map[string]interface{}:
		return rr.resolveMap(typedData, currentPath, newStringSet(alreadyResolved))
	case []interface{}:
		return rr.resolveList(typedData, currentPath, newStringSet(alreadyResolved))
	}
	return data, nil
}

// resolveMap is a helper function which can recursively resolve a map
func (rr *ReferenceResolver) resolveMap(data map[string]interface{}, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	isRef, uri, err := checkForReference(data, currentPath)
	if err != nil {
		return err, nil
	}

	if isRef {
		// current map is a reference
		sub, err := rr.resolveReference(uri, currentPath, alreadyResolved)
		if err != nil {
			return nil, fmt.Errorf("error resolving reference at %s: %w", currentPath.Child(gojsonschema.KEY_REF).String(), err)
		}
		return sub, nil
	}

	// current map is not a reference
	// iterate over entries and resolve each of them
	res := map[string]interface{}{}
	for k, v := range data {
		subPath := currentPath.Child(k)
		sub, err := rr.resolve(v, subPath, alreadyResolved)
		if err != nil {
			return nil, err
		}
		res[k] = sub
	}
	return res, nil
}

// checkForReference checks whether a given map is a valid reference
// If it is a reference, it returns true and the URL of the reference.
// Otherwise, it returns false and an empty string.
func checkForReference(data map[string]interface{}, currentPath *field.Path) (bool, string, error) {
	value, ok := data[gojsonschema.KEY_REF]
	if !ok {
		// no reference
		return false, "", nil
	}
	typedValue, ok := value.(string)
	if !ok {
		return true, "", fmt.Errorf("invalid reference value at %s: expected string, got %v", currentPath.Child(gojsonschema.KEY_REF).String(), value)
	}
	return true, typedValue, nil
}

// resolveList is a helper function which can recursively resolve a list
func (rr *ReferenceResolver) resolveList(data []interface{}, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	resList := make([]interface{}, len(data))
	for i, e := range data {
		sub, err := rr.resolve(e, currentPath.Index(i), alreadyResolved)
		if err != nil {
			return nil, err
		}
		resList[i] = sub
	}
	return resList, nil
}

// resolveReference resolves a reference
func (rr *ReferenceResolver) resolveReference(s string, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	uri, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	refID := absoluteRef(rr.ComponentVersion, s)
	if alreadyResolved.contains(refID) {
		return nil, fmt.Errorf("cyclic references detected: reference %q from component %s:%s is part of a cycle", s, rr.ComponentVersion.GetName(), rr.ComponentVersion.GetVersion())
	}
	alreadyResolved.add(refID)
	switch uri.Scheme {
	case "local":
		return rr.handleLocalReference(uri, currentPath, alreadyResolved)
	case "blueprint":
		return rr.handleBlueprintReference(uri, currentPath, alreadyResolved)
	case "cd":
		return rr.handleComponentDescriptorReference(uri, currentPath, alreadyResolved)
	}

	// unknown reference scheme
	// rebuild reference because it is replaced in calling method
	return map[string]interface{}{
		gojsonschema.KEY_REF: s,
	}, nil
}

func (rr *ReferenceResolver) handleLocalReference(uri *url.URL, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	if len(uri.Path) != 0 {
		return nil, errors.New("a path is not supported for local resources")
	}
	schema, ok := rr.LocalTypes[uri.Host]
	if !ok {
		return nil, fmt.Errorf("type %s is not defined in local types", uri.Host)
	}
	data, err := decodeJSON(schema.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json into go struct: %w", err)
	}
	res, err := rr.resolve(data, currentPath, alreadyResolved)
	if err != nil {
		return nil, err
	}
	return resolveFragment(uri, res)
}

func (rr *ReferenceResolver) handleBlueprintReference(uri *url.URL, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	if rr.BlueprintFs == nil {
		return nil, errors.New("no filesystem defined to read a local schema")
	}
	filePath := filepath.Join(uri.Host, uri.Path)
	schemaBytes, err := vfs.ReadFile(rr.BlueprintFs, filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read local schema from %s: %w", filePath, err)
	}
	data, err := decodeJSON(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json into go struct: %w", err)
	}
	res, err := rr.resolve(data, currentPath, alreadyResolved)
	if err != nil {
		return nil, err
	}
	return resolveFragment(uri, res)
}

func (rr *ReferenceResolver) handleComponentDescriptorReference(uri *url.URL, currentPath *field.Path, alreadyResolved stringSet) (interface{}, error) {
	if rr.ComponentVersion == nil {
		return nil, errors.New("no component descriptor defined to resolve the ref")
	}
	if rr.RegistryAccess == nil {
		return nil, errors.New("no component reference resolver defined to resolve the ref")
	}
	cdUri, err := cdutils.ParseURI(uri.String())
	if err != nil {
		return nil, err
	}
	repositoryContext := rr.RepositoryContext
	if repositoryContext == nil {
		repositoryContext = rr.ComponentVersion.GetRepositoryContext()
	}
	cd, resource, err := cdUri.GetResource(rr.ComponentVersion, repositoryContext)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()

	resourceContent, err := resource.GetTypedContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch jsonschema for '%s': %w", uri.String(), err)
	}
	result, ok := resourceContent.Resource.([]byte)
	if !ok {
		return nil, fmt.Errorf("received resource of type %T but expected type []byte", result)
	}
	data, err := decodeJSON(result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json into go struct: %w", err)
	}

	resolved, err := NewReferenceResolver(&ReferenceContext{
		LocalTypes:        nil,
		BlueprintFs:       nil,
		ComponentVersion:  cd,
		RegistryAccess:    rr.RegistryAccess,
		RepositoryContext: rr.RepositoryContext,
	}).resolve(data, currentPath, alreadyResolved)
	if err != nil {
		return nil, err
	}
	return resolveFragment(uri, resolved)
}

// decodeJSON decodes a json string into go structs
func decodeJSON(rawData []byte) (interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(rawData))
	decoder.UseNumber()
	var data interface{}
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// resolveFragment is used to resolve references which don't refer to a complete resource, but to a specific path within it
func resolveFragment(uri *url.URL, data interface{}) (interface{}, error) {
	if len(uri.Fragment) == 0 {
		return data, nil
	}
	fragment := strings.TrimPrefix(uri.Fragment, "/")
	fragment = strings.TrimSuffix(fragment, "/")
	frags := strings.Split(fragment, "/")
	path := field.NewPath("")
	current := data
	for _, f := range frags {
		elem, ok := current.(map[string]interface{})
		if !ok {
			return data, fmt.Errorf("unable to resolve fragment %s: element at %q is no object", uri.Fragment, path.String())
		}
		current, ok = elem[f]
		if !ok {
			return data, fmt.Errorf("error resolving fragment %s: element at %q doesn't have a field %q", uri.Fragment, path.String(), f)
		}
		path = path.Child(f)
	}
	return current, nil
}

// auxiliary type
type stringSet map[string]struct{}

func (s stringSet) contains(key string) bool {
	_, ok := s[key]
	return ok
}

func (s stringSet) add(key string) {
	s[key] = struct{}{}
}

// absoluteRef prefixes a given refstring with name and version of the component descriptor where it came from
// this transforms the relative references into an absolute identifier
func absoluteRef(cd model.ComponentVersion, ref string) string {
	if cd == nil {
		return ref
	}
	return fmt.Sprintf("%s:%s::%s", cd.GetName(), cd.GetVersion(), ref)
}

func newStringSet(old stringSet) stringSet {
	res := stringSet{}
	for k := range old {
		res.add(k)
	}
	return res
}
