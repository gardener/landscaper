// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/landscaper/pkg/apis/core/install"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils"
)

// LocalAccessType is the name of the local access type
const LocalAccessType = "local"

func init() {
	cdv2.KnownAccessTypes[LocalAccessType] = LocalAccessCodec
}

// LocalAccess describes the local access for a landscaper blueprint
type LocalAccess struct {
	cdv2.ObjectType `json:",inline"`
}

var _ cdv2.TypedObjectAccessor = &LocalAccess{}

// GetData is the noop implementation for a local accessor
func (l LocalAccess) GetData() ([]byte, error) {
	return []byte{}, nil
}

// SetData is the noop implementation for a local accessor
func (l LocalAccess) SetData(bytes []byte) error { return nil }

// LocalAccessCodec implements the acccess codec for the local accessor.
var LocalAccessCodec = &cdv2.TypedObjectCodecWrapper{
	TypedObjectDecoder: cdv2.TypedObjectDecoderFunc(func(data []byte) (cdv2.TypedObjectAccessor, error) {
		var localAccess LocalAccess
		if err := json.Unmarshal(data, &localAccess); err != nil {
			return nil, err
		}
		return &localAccess, nil
	}),
	TypedObjectEncoder: cdv2.TypedObjectEncoderFunc(func(accessor cdv2.TypedObjectAccessor) ([]byte, error) {
		localAccess, ok := accessor.(*LocalAccess)
		if !ok {
			return nil, fmt.Errorf("accessor is not of type %s", LocalAccessType)
		}
		return json.Marshal(localAccess)
	}),
}

/**

definition.yaml
blob.yaml --> url: xxx

*/

// BlueprintFileName is the filename of a component definition on a local path
const BlueprintFileName = "blueprint.yaml"

type localRegistry struct {
	log   logr.Logger
	paths []string

	decoder runtime.Decoder

	Index Index
}

// NewLocalRegistry creates a new ociRegistry that serves local blueprints.
func NewLocalRegistry(log logr.Logger, paths ...string) (*localRegistry, error) {
	lsScheme := runtime.NewScheme()
	if err := install.AddToScheme(lsScheme); err != nil {
		return nil, err
	}

	r := &localRegistry{
		log:     log,
		paths:   paths,
		decoder: serializer.NewCodecFactory(lsScheme).UniversalDecoder(),
		Index:   Index{},
	}

	if err := r.setDefinitions(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *localRegistry) setDefinitions() error {
	for _, path := range r.paths {
		index, err := r.findDefinitionsInPath(path)
		if err != nil {
			return err
		}
		r.Index.Merge(index)
	}
	return nil
}

// findDefinitionsInPath walks the given path and tries to parse each file with the BlueprintFileName.
// The component definition's directory is set as its corresponding blob.
func (r *localRegistry) findDefinitionsInPath(path string) (Index, error) {
	index := Index{}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			r.log.Error(err, "unable to walk path", "path", path)
			return nil
		}

		if info.Name() != BlueprintFileName {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to read file %s: %w", path, err)
		}

		definition := &v1alpha1.Blueprint{}
		if _, _, err := r.decoder.Decode(data, nil, definition); err != nil {
			return fmt.Errorf("unable to decode blueprint from file %s: %w", path, err)
		}

		index.Add(LocalBlueprintReference{
			SourcePath: path,
			Blueprint:  definition,
			blobPath:   filepath.Dir(path),
		})

		return nil
	})
	return index, err
}

// GetDefinition returns the definition for a specific name, version and type.
func (r *localRegistry) GetBlueprint(_ context.Context, ref cdv2.Resource) (*v1alpha1.Blueprint, error) {
	var (
		name    = ref.GetName()
		version = ref.GetVersion()
	)

	if _, ok := r.Index[name]; !ok {
		return nil, NewComponentNotFoundError(name, nil)
	}
	intRef, ok := r.Index[name][version]
	if !ok {
		return nil, NewVersionNotFoundError(name, version, nil)
	}
	return intRef.Blueprint, nil
}

// GetBlob returns the blob content for a component definition.
func (r *localRegistry) GetContent(ctx context.Context, ref cdv2.Resource, fs vfs.FileSystem) error {
	var (
		name    = ref.GetName()
		version = ref.GetVersion()
	)

	if _, ok := r.Index[name]; !ok {
		return NewComponentNotFoundError(name, nil)
	}
	intRef, ok := r.Index[name][version]
	if !ok {
		return NewVersionNotFoundError(name, version, nil)
	}

	blobFS, err := projectionfs.New(osfs.New(), intRef.blobPath)
	if err != nil {
		return err
	}

	return utils.CopyFS(blobFS, fs, "/", "/")
}

// BlueprintReferenceTemplate is the reference to a local definition
type LocalBlueprintReference struct {
	SourcePath string
	Blueprint  *v1alpha1.Blueprint
	blobPath   string
}

// Index is internal Index structure for definition.
// The map indexes name and secondly the version
/**
name:
	version:
		sourcePath: path
		definition: def
		blobPath: def
*/
type Index map[string]map[string]LocalBlueprintReference

// Add adds or updates a definition reference in the Index.
func (i Index) Add(ref LocalBlueprintReference) {
	def := ref.Blueprint
	if _, ok := i[def.Name]; !ok {
		i[def.Name] = make(map[string]LocalBlueprintReference)
	}

	i[def.Name][def.Version] = ref
}

// Merge merges the Index a in into the current Index.
// Whereas the keys of Index a overwrite similar keys of the current Index.
func (i Index) Merge(a Index) {
	if len(a) == 0 {
		return
	}

	for name, versionedDefinitions := range a {
		if _, ok := i[name]; !ok {
			i[name] = versionedDefinitions
			continue
		}

		for version, def := range versionedDefinitions {
			i[name][version] = def
		}
	}
}
