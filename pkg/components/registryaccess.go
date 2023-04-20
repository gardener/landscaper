package components

import (
	"context"

	"github.com/gardener/component-cli/ociclient/cache"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-errors/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
	ocmadapter "github.com/gardener/landscaper/pkg/components/ocm"
)

func NewRegistryAccess(ctx context.Context, componentModelVersion string, secrets []corev1.Secret, sharedCache cache.Cache,
	localRegistryConfig *config.LocalRegistryConfiguration, ociRegistryConfig *config.OCIConfiguration,
	inlineCd *cdv2.ComponentDescriptor) (model.RegistryAccess, error) {

	if componentModelVersion == cnudie.ComponentModelVersion {
		return cnudie.NewCnudieRegistry(ctx, secrets, sharedCache,
			localRegistryConfig, ociRegistryConfig,
			inlineCd)
	} else if componentModelVersion == ocmadapter.ComponentModelVersion {
		return ocmadapter.NewOCMRegistry(ctx, secrets,
			localRegistryConfig, ociRegistryConfig,
			inlineCd)
	}
	return nil, errors.Errorf("The version of the component model has to be specified. Thus, either v2 (=cnudie) or v3 (=ocm)")
}
