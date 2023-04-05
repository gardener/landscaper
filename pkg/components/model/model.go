package model

import (
	"context"
	"io"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Registry Access hat eine Sammlung von Registries, um die ComponentVersion (UND Ressourcen zu holen)
type RegistryAccess interface {
	GetComponentVersion(ctx context.Context, cdRef *lsv1alpha1.ComponentDescriptorReference) (ComponentVersion, error)

	//GetResource
	//GetResource(ctx context.Context, )

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

// Muss eine Reihe von Resolvern haben, um die Blobs von verschiedenen Typen abzuholen
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
