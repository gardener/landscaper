// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactsregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/oci"
	"github.com/gardener/landscaper/pkg/utils/oci/cache"
)

type local struct {
	fs    vfs.FileSystem
	paths []string
	cache cache.Cache
}

var _ cache.InjectCache = &local{}

// NewLocalRegistry creates a new local registry.
func NewLocalRegistry(fs vfs.FileSystem, paths ...string) TypedRegistry {
	return &local{
		fs:    fs,
		paths: paths,
	}
}

func (l *local) InjectCache(c cache.Cache) error {
	l.cache = c
	return nil
}

func (l *local) GetBlob(_ context.Context, access cdv2.TypedObjectAccessor, writer io.Writer) (string, error) {
	if access.GetType() != l.Type() {
		return "", fmt.Errorf("wrong access type '%s' expected '%s'", access.GetType(), l.Type())
	}
	localAccess := access.(*LocalAccess)

	if len(localAccess.Path) != 0 {
		// directly read the given file/directory
		info, err := l.fs.Stat(localAccess.Path)
		if err != nil {
			return "", err
		}

		if !info.IsDir() {
			file, err := l.fs.Open(localAccess.Path)
			if err != nil {
				return "", err
			}
			defer file.Close()
			if _, err := io.Copy(writer, file); err != nil {
				return "", err
			}
			return "", nil
		}

		// directories are returned as tar
		fs, err := projectionfs.New(osfs.New(), localAccess.Path)
		if err != nil {
			return "", err
		}
		if err := utils.BuildTar(fs, "/", writer); err != nil {
			return "", err
		}
		return oci.MediaTypeTar, nil
	}

	return "", errors.New("currently only access types with the path is supported")
}

func (l *local) Type() string {
	return LocalAccessType
}

// LocalAccessType is the name of the local access type
const LocalAccessType = "local"

func init() {
	cdv2.KnownAccessTypes[LocalAccessType] = LocalAccessCodec
}

// LocalAccess describes the local access for a landscaper blueprint
type LocalAccess struct {
	cdv2.ObjectType `json:",inline"`
	// +optional
	Path string `json:"path,omitempty"`
}

var _ cdv2.TypedObjectAccessor = &LocalAccess{}

// GetData is the noop implementation for a local accessor
func (l LocalAccess) GetData() ([]byte, error) {
	return json.Marshal(l)
}

// SetData is the noop implementation for a local accessor
func (l *LocalAccess) SetData(bytes []byte) error {
	var newLocalAccess LocalAccess
	if err := yaml.Unmarshal(bytes, &newLocalAccess); err != nil {
		return err
	}
	l.Path = newLocalAccess.Path
	return nil
}

// LocalAccessCodec implements the acccess codec for the local accessor.
var LocalAccessCodec = &cdv2.TypedObjectCodecWrapper{
	TypedObjectDecoder: cdv2.TypedObjectDecoderFunc(func(data []byte) (cdv2.TypedObjectAccessor, error) {
		var localAccess LocalAccess
		if err := yaml.Unmarshal(data, &localAccess); err != nil {
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
