package model

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type Registry interface {
	GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (ComponentVersion, error)

	// temporary
	GetComponentResolver() ctf.ComponentResolver
}

type ComponentVersion interface {
	GetDescriptor() ([]byte, error)
	GetDependency(name string) (ComponentVersion, error)
	GetResource(name string, identity map[string]string) (Resource, error)
}

type Resource interface {
	GetDescriptor() ([]byte, error)
	GetBlob() ([]byte, error)
	GetBlobInfo() (BlobInfo, error)
}

type BlobInfo struct {
	MediaType string
	Digest    string
}
