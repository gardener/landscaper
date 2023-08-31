package ocmlib

import (
	"context"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/oci"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/helm"
	"github.com/open-component-model/ocm/pkg/helm/loader"
	"github.com/open-component-model/ocm/pkg/runtime"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
	_ "github.com/gardener/landscaper/pkg/components/ocmlib/resourcetypehandlers"
)

type Resource struct {
	resourceAccess  ocm.ResourceAccess
	handlerRegistry *registries.ResourceHandlerRegistry
}

func NewResource(access ocm.ResourceAccess) model.Resource {
	return &Resource{
		resourceAccess:  access,
		handlerRegistry: registries.Registry,
	}
}

func (r *Resource) GetName() string {
	return r.resourceAccess.Meta().GetName()
}

func (r *Resource) GetVersion() string {
	return r.resourceAccess.Meta().GetVersion()
}

func (r *Resource) GetType() string {
	return r.resourceAccess.Meta().GetType()
}

func (r *Resource) GetAccessType() string {
	spec, err := r.resourceAccess.Access()
	if err != nil {
		return ""
	}
	return spec.GetType()
}

func (r *Resource) GetResource() (*types.Resource, error) {
	spec := r.resourceAccess.Meta()
	data, err := runtime.DefaultYAMLEncoding.Marshal(spec)
	if err != nil {
		return nil, err
	}

	lsspec := types.Resource{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsspec)
	if err != nil {
		return nil, err
	}

	return &lsspec, err
}

func (r *Resource) GetTypedContent(ctx context.Context) (*model.TypedResourceContent, error) {
	handler := r.handlerRegistry.Get(r.GetType())
	return handler.GetResourceContent(ctx, r, r.resourceAccess)
}

func (r *Resource) GetCachingIdentity(ctx context.Context) string {
	spec, err := r.resourceAccess.Access()
	if err != nil {
		return ""
	}
	return spec.GetInexpensiveContentVersionIdentity(r.resourceAccess.ComponentVersion())
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type HelmChartProvider struct {
	ocictx  oci.Context
	ref     string
	version string
	repourl string
}

func (h *HelmChartProvider) GetTypedContent(ctx context.Context) (_ *model.TypedResourceContent, rerr error) {
	access, err := helm.DownloadChart(common.NewPrinter(nil), h.ocictx, h.ref, h.version, h.repourl)
	if err != nil {
		return nil, err
	}
	defer errors.PropagateError(&rerr, access.Close)

	chartLoader := loader.AccessLoader(access)
	helmChart, err := chartLoader.Chart()
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     types.HelmChartResourceType,
		Resource: helmChart,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
