package model

import (
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type Registry interface {
	GetComponentVersion(cdRef *lsv1alpha1.ComponentDescriptorReference) (ComponentVersion, error)

	// temporary
	GetComponentResolver() ctf.ComponentResolver
}

type ComponentVersion interface {
	GetDescriptor() ([]byte, error)
	GetDependency(name string) (ComponentVersion, error)
	GetResource(name string) (Resource, error)
}

type Resource interface {
	GetDescriptor() ([]byte, error)
	GetData() ([]byte, error)
}
