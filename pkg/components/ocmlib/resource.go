// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ocmlib

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/goutils/errors"

	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/tech/helm"
	"ocm.software/ocm/api/tech/helm/loader"
	common "ocm.software/ocm/api/utils/misc"
	"ocm.software/ocm/api/utils/runtime"

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

	accessSpec, err := r.resourceAccess.Access()
	if err != nil {
		return nil, err
	}
	accessData, err := runtime.DefaultJSONEncoding.Marshal(accessSpec)
	if err != nil {
		return nil, err
	}
	lsspec := types.Resource{}
	err = runtime.DefaultYAMLEncoding.Unmarshal(data, &lsspec)
	if err != nil {
		return nil, err
	}

	lsspec.Access = &cdv2.UnstructuredTypedObject{}
	err = lsspec.Access.UnmarshalJSON(accessData)
	if err != nil {
		return nil, err
	}

	return &lsspec, err
}

func (r *Resource) GetTypedContent(ctx context.Context) (*model.TypedResourceContent, error) {
	handler := r.handlerRegistry.Get(r.GetType())
	if handler != nil {
		return handler.GetResourceContent(ctx, r, r.resourceAccess)
	}
	return nil, fmt.Errorf("no handler found for resource type %s", r.GetType())
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
