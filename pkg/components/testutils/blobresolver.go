package testutils

import (
	"context"
	"io"

	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/landscaper/pkg/components/model/types"
)

type testBlobResolver struct {
	reader io.Reader
	info   *types.BlobInfo
}

var _ ctf.TypedBlobResolver = &testBlobResolver{}

func newTestBlobResolverFromReader(blobReader io.Reader, blobInfo *types.BlobInfo) ctf.BlobResolver {
	return &testBlobResolver{
		reader: blobReader,
		info:   blobInfo,
	}
}

func (b testBlobResolver) CanResolve(_ types.Resource) bool {
	return true
}

func (b testBlobResolver) Info(ctx context.Context, res types.Resource) (*types.BlobInfo, error) {
	return b.info, nil
}

func (b testBlobResolver) Resolve(ctx context.Context, res types.Resource, writer io.Writer) (*types.BlobInfo, error) {
	if _, err := io.Copy(writer, b.reader); err != nil {
		return nil, err
	}
	return b.info, nil
}
