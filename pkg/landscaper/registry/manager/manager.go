package manager

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"

	"github.com/gardener/landscaper/pkg/apis/config"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	regapi "github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/landscaper/registry/oci"
)

// Interface describes the interface for a regapi manager.
// A regapi manager exposes itself a regapi api and delegates the
// request to the specific implementations.
type Interface interface {
	regapi.Registry
	AddRegistry(schemaName string, scheme cdv2.AccessCodec, registry regapi.Registry) error
}

// NewRegistryManager creates a new regapi manager or the given regapi configuration
func NewRegistryManager(log logr.Logger, config *config.RegistryConfiguration) (Interface, error) {
	registries := registries{}

	if config.Local != nil {
		local, err := regapi.NewLocalRegistry(log, config.Local.Paths)
		if err != nil {
			return nil, fmt.Errorf("unable to setup local regapi: %w", err)
		}
		if err := registries.AddRegistry(regapi.LocalAccessType, regapi.LocalAccessCodec, local); err != nil {
			return nil, err
		}
	}

	if config.OCI != nil {
		ociReg, err := oci.New(log, config.OCI)
		if err != nil {
			return nil, fmt.Errorf("unable to setup oci regapi: %w", err)
		}
		if err := registries.AddRegistry(cdv2.OCIRegistryType, cdv2.KnownAccessTypes[cdv2.OCIRegistryType], ociReg); err != nil {
			return nil, err
		}
	}

	return registries, nil
}

type registries map[string]regapi.Registry

var _ Interface = registries{}

func (r registries) AddRegistry(schemaName string, scheme cdv2.AccessCodec, registry regapi.Registry) error {
	cdv2.KnownAccessTypes[schemaName] = scheme
	r[schemaName] = registry
	return nil
}

func (r registries) GetBlueprint(ctx context.Context, ref cdv2.Resource) (*v1alpha1.Blueprint, error) {
	reg, ok := r[ref.Access.GetType()]
	if !ok {
		return nil, regapi.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetBlueprint(ctx, ref)
}

func (r registries) GetContent(ctx context.Context, ref cdv2.Resource) (afero.Fs, error) {
	reg, ok := r[ref.Access.GetType()]
	if !ok {
		return nil, regapi.NewWrongTypeError(ref.Access.GetType(), ref.Name, ref.Version, nil)
	}
	return reg.GetContent(ctx, ref)
}
