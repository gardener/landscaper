package cd_facade

import (
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Cache interface {
	io.Closer
	Get(desc ocispecv1.Descriptor) (io.ReadCloser, error)
	Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error
}
