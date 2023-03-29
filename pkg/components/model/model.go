package model

import (
	"context"
	"io"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type Registry interface {
	GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (ComponentVersion, error)

	// temporary
	GetComponentResolver() ctf.ComponentResolver
}

type ComponentVersion interface {
	GetName() string
	GetVersion() string
	GetRepositoryContext() []byte
	GetDescriptor(ctx context.Context) ([]byte, error)
	GetDependency(ctx context.Context, name string) (ComponentVersion, error)
	GetResource(name string, identity map[string]string) (Resource, error)
}

type Resource interface {
	GetName() string
	GetVersion() string
	GetDescriptor(ctx context.Context) ([]byte, error)
	GetBlob(ctx context.Context, writer io.Writer) error
	GetBlobInfo(ctx context.Context) (*BlobInfo, error)
}

type BlobInfo struct {
	MediaType string
	Digest    string
}
