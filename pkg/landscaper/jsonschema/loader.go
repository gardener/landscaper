// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jsonschema

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/xeipuuv/gojsonreference"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// JSONSchemaMediaType is the custom media type for a jsonschema in a oci registry
const JSONSchemaMediaType = "application/json+jsonschema"

// LoaderWrapper wraps a JSONLoader with a landscaper loader.
type LoaderWrapper struct {
	LoaderConfig
	gojsonschema.JSONLoader
}

func NewWrappedLoader(config LoaderConfig, loader gojsonschema.JSONLoader) gojsonschema.JSONLoader {
	if config.DefaultLoader == nil {
		config.DefaultLoader = loader
	}
	return &LoaderWrapper{
		LoaderConfig: config,
		JSONLoader:   loader,
	}
}

func (l LoaderWrapper) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return &LoaderFactory{
		LoaderConfig: l.LoaderConfig,
	}
}

// Loader is the landscaper specific jsonscheme loader.
// It resolves referecens of type: local, blueprint and cd.
type Loader struct {
	LoaderConfig
	source string
}

// LoaderConfig is the landscaper specific laoder configuration
// to resolve landscaper specific schema refs.
type LoaderConfig struct {
	// LocalTypes is a map of blueprint locally defined types.
	// It is a map of schema name to schema definition
	LocalTypes map[string]lsv1alpha1.JSONSchemaDefinition
	// BlueprintFs is the virtual filesystem that is used to resolve "blueprint" refs
	BlueprintFs vfs.FileSystem
	// ComponentDescriptor contains the current blueprint's component descriptor.
	ComponentDescriptor *cdv2.ComponentDescriptor
	// ComponentReferenceResolver is a function that resolves component references
	ComponentReferenceResolver cdutils.ResolveComponentReferenceFunc
	// BlobResolver is the registry to resolve resources of the component descriptor.
	BlobResolver ctf.BlobResolver
	// DefaultLoader is the fallback loader that is used of the protocol is unknown.
	DefaultLoader gojsonschema.JSONLoader
}

// LoaderFactory is the factory that creates a new landscaper specific loader.
type LoaderFactory struct {
	LoaderConfig
}

func (l LoaderFactory) New(source string) gojsonschema.JSONLoader {
	return &Loader{
		LoaderConfig: l.LoaderConfig,
		source:       source,
	}
}

var _ gojsonschema.JSONLoader = &Loader{}

func (l Loader) JsonSource() interface{} {
	return l.source
}

func (l *Loader) LoadJSON() (interface{}, error) {
	var err error

	reference, err := l.JsonReference()
	if err != nil {
		return nil, err
	}

	refURL := reference.GetUrl()
	var schemaJSONBytes []byte
	switch refURL.Scheme {
	case "local":
		schemaJSONBytes, err = l.loadLocalReference(refURL)
	case "blueprint":
		schemaJSONBytes, err = l.loadBlueprintReference(refURL)
	case "cd":
		schemaJSONBytes, err = l.loadComponentDescriptorReference(refURL)
	default:
		if l.DefaultLoader == nil {
			return nil, fmt.Errorf("unsupported ref %s", refURL.String())
		}
		return l.DefaultLoader.LoaderFactory().New(l.source).LoadJSON()
	}
	if err != nil {
		return nil, err
	}

	if err := ValidateSchema(schemaJSONBytes); err != nil {
		return nil, err
	}

	var schemaJSON interface{}
	if err := yaml.Unmarshal(schemaJSONBytes, &schemaJSON); err != nil {
		return nil, err
	}
	return schemaJSON, nil
}

func (l Loader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference(l.JsonSource().(string))
}

func (l Loader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return &LoaderFactory{
		LoaderConfig: l.LoaderConfig,
	}
}

func (l *Loader) loadLocalReference(refURL *url.URL) ([]byte, error) {
	if len(refURL.Path) != 0 {
		return nil, errors.New("a path is not supported for local resources")
	}
	schemaBytes, ok := l.LocalTypes[refURL.Host]
	if !ok {
		return nil, fmt.Errorf("type %s is not defined in local types", refURL.Host)
	}
	return schemaBytes, nil
}

func (l *Loader) loadBlueprintReference(refURL *url.URL) ([]byte, error) {
	if l.BlueprintFs == nil {
		return nil, errors.New("no filesystem defined to read a local schema")
	}
	filePath := filepath.Join(refURL.Host, refURL.Path)
	schemaBytes, err := vfs.ReadFile(l.BlueprintFs, filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read local schema from %s: %w", filePath, err)
	}
	return schemaBytes, nil
}

func (l *Loader) loadComponentDescriptorReference(refURL *url.URL) ([]byte, error) {
	if l.ComponentDescriptor == nil {
		return nil, errors.New("no component descriptor defined to resolve the ref")
	}
	if l.ComponentReferenceResolver == nil {
		return nil, errors.New("no component reference resolver defined to resolve the ref")
	}
	if l.BlobResolver == nil {
		return nil, errors.New("no blobl resolver defined to resolve the ref")
	}
	uri, err := cdutils.ParseURI(refURL.String())
	if err != nil {
		return nil, err
	}
	kind, res, err := uri.Get(l.ComponentDescriptor, l.ComponentReferenceResolver)
	if err != nil {
		return nil, err
	}
	if kind == lsv1alpha1.ComponentResourceKind {
		return nil, fmt.Errorf("expected a resource but reference resolves to a component")
	}
	resource := res.(cdv2.Resource)

	ctx := context.Background()
	defer ctx.Done()
	var JSONSchemaBuf bytes.Buffer
	_, err = l.BlobResolver.Resolve(ctx, resource, &JSONSchemaBuf)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch jsonschema for '%s': %w", refURL.String(), err)
	}

	return JSONSchemaBuf.Bytes(), nil
}
