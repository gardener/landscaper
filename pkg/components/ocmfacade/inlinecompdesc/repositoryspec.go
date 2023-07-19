package inlinecompdesc

//import (
//	"github.com/open-component-model/ocm/pkg/common/accessio"
//	"github.com/open-component-model/ocm/pkg/contexts/credentials"
//	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
//	"github.com/open-component-model/ocm/pkg/runtime"
//)
//
//const (
//	Type   = "landscaper.gardener.cloud/local"
//	TypeV1 = Type + runtime.VersionSeparator + "v1"
//)
//
//type RepositorySpec struct {
//	runtime.ObjectVersionedType `json:",inline"`
//	accessio.StandardOptions    `json:",omitempty"`
//	CompDescDirPath             string `json:"componentDescriptorDirPath"`
//	BlobDirPath                 string `json:"blobDirPath"`
//}
//
//var (
//	_ accessio.Option    = (*RepositorySpec)(nil)
//	_ cpi.RepositorySpec = (*RepositorySpec)(nil)
//)
//
//func NewRepositorySpec(compDescDirPath string, blobDirPath string, opts ...accessio.Option) (*RepositorySpec, error) {
//	spec := &RepositorySpec{
//		ObjectVersionedType: runtime.NewVersionedObjectType(Type),
//		CompDescDirPath:     compDescDirPath,
//		BlobDirPath:         blobDirPath,
//	}
//	_, err := accessio.AccessOptions(&spec.StandardOptions, opts...)
//	if err != nil {
//		return nil, err
//	}
//	return spec, nil
//}
//
//func (r *RepositorySpec) AsUniformSpec(ctx cpi.Context) *cpi.UniformRepositorySpec {
//	return nil
//}
//
//func (r *RepositorySpec) Repository(ctx cpi.Context, credentials credentials.Credentials) (cpi.Repository, error) {
//	return NewRepository(ctx, r)
//}
