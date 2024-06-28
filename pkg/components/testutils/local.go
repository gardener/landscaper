// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"bytes"
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	apiconfig "github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/tar"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
)

func NewLocalRegistryAccess(ctx context.Context, rootPath string) (model.RegistryAccess, error) {
	repositoryContext := &cdv2.UnstructuredTypedObject{}
	if err := repositoryContext.UnmarshalJSON([]byte(`{"type": "local","filePath": "./"}`)); err != nil {
		return nil, err
	}
	return registries.GetFactory(true).NewRegistryAccess(ctx, &model.RegistryAccessOptions{
		LocalRegistryConfig: &apiconfig.LocalRegistryConfiguration{RootPath: rootPath},
		AdditionalRepositoryContexts: []types.PrioritizedRepositoryContext{
			{
				RepositoryContext: repositoryContext,
				Priority:          10,
			},
		},
	})
}

////////////////////////////////////////////////////////////

// LocalRepositoryType defines the local repository context type.
const LocalRepositoryType = "local"

func NewLocalFilesystemBlobAccess(path, mediaType string) (cdv2.UnstructuredTypedObject, error) {
	return cdv2.NewUnstructured(cdv2.NewLocalFilesystemBlobAccess(path, mediaType))
}

// LocalRepository describes a local repository
type LocalRepository struct {
	cdv2.ObjectType
	BaseURL string `json:"baseUrl"`
}

// NewLocalRepository creates a new local repository
func NewLocalRepository(baseUrl string) *LocalRepository {
	return &LocalRepository{
		ObjectType: cdv2.ObjectType{
			Type: LocalRepositoryType,
		},
		BaseURL: baseUrl,
	}
}

func NewLocalRepositoryContext(baseURL string) (cdv2.UnstructuredTypedObject, error) {
	return cdv2.NewUnstructured(NewLocalRepository(baseURL))
}

// LocalFilesystemBlobResolver implements the BlobResolver interface for
// "localFilesystemBlob" access types.
type LocalFilesystemBlobResolver struct {
	BaseFilesystemBlobResolver
}

// NewLocalFilesystemBlobResolver creates a new local filesystem blob resolver.
func NewLocalFilesystemBlobResolver(fs vfs.FileSystem) *LocalFilesystemBlobResolver {
	return &LocalFilesystemBlobResolver{
		BaseFilesystemBlobResolver: BaseFilesystemBlobResolver{fs: fs},
	}
}

func (ca *LocalFilesystemBlobResolver) CanResolve(resource types.Resource) bool {
	return resource.Access != nil && resource.Access.GetType() == cdv2.LocalFilesystemBlobType
}

func (ca *LocalFilesystemBlobResolver) Info(_ context.Context, res types.Resource) (*ctf.BlobInfo, error) {
	info, file, err := ca.resolve(res)
	if err != nil {
		return nil, err
	}
	if file != nil {
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return info, nil
}

// Resolve fetches the blob for a given resource and writes it to the given tar.
func (ca *LocalFilesystemBlobResolver) Resolve(_ context.Context, res types.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	info, file, err := ca.resolve(res)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(writer, file); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return info, nil
}

func (ca *LocalFilesystemBlobResolver) resolve(res types.Resource) (*ctf.BlobInfo, io.ReadCloser, error) {
	if res.Access == nil || res.Access.GetType() != cdv2.LocalFilesystemBlobType {
		return nil, nil, ctf.UnsupportedResolveType
	}

	localFSAccess := &cdv2.LocalFilesystemBlobAccess{}
	if err := cdv2.NewCodec(nil, nil, nil).Decode(res.Access.Raw, localFSAccess); err != nil {
		return nil, nil, fmt.Errorf("unable to decode access to type '%s': %w", res.Access.GetType(), err)
	}

	blobpath := ctf.BlobPath(localFSAccess.Filename)

	info, file, err := ca.ResolveFromFs(blobpath)
	if err != nil {
		return nil, nil, err
	}
	info.MediaType = res.Type
	if len(localFSAccess.MediaType) != 0 {
		info.MediaType = localFSAccess.MediaType
	}
	return info, file, nil
}

// BaseFilesystemBlobResolver implements a common method for filesystem.
type BaseFilesystemBlobResolver struct {
	fs vfs.FileSystem
}

// ResolveFromFs resolves a blob from a given path.
func (res *BaseFilesystemBlobResolver) ResolveFromFs(blobpath string) (*ctf.BlobInfo, io.ReadCloser, error) {
	info, err := res.fs.Stat(blobpath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get fileinfo for %s: %w", blobpath, err)
	}
	if info.IsDir() {
		var data bytes.Buffer
		if err := tar.BuildTarGzip(res.fs, blobpath, &data); err != nil {
			return nil, nil, fmt.Errorf("unable to build tar gz: %w", err)
		}
		return &ctf.BlobInfo{
			MediaType: "",
			Digest:    digest.FromBytes(data.Bytes()).String(),
			Size:      int64(data.Len()),
		}, io.NopCloser(&data), nil
	}
	file, err := res.fs.Open(blobpath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open blob from %s", blobpath)
	}

	dig, err := digest.FromReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate dig from %s: %w", blobpath, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, fmt.Errorf("unable to reset file reader: %w", err)
	}
	return &ctf.BlobInfo{
		MediaType: "",
		Digest:    dig.String(),
		Size:      info.Size(),
	}, file, nil
}
