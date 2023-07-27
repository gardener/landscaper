package blueprint

import (
	"context"

	"github.com/mandelsoft/filepath/pkg/filepath"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmlib/registries"
)

func init() {
	registries.Registry.Register(mediatype.BlueprintType, New())
	registries.Registry.Register(mediatype.OldBlueprintType, New())
}

type BlueprintHandler struct {
	cache *blueprint.Store
}

func New() *BlueprintHandler {
	return &BlueprintHandler{
		cache: blueprint.GetBlueprintStore(),
	}
}
func (h *BlueprintHandler) GetResourceContent(ctx context.Context, r model.Resource, access ocm.ResourceAccess) (*model.TypedResourceContent, error) {
	res, err := h.cache.Get(ctx, r.GetCachingIdentity(ctx))
	if err != nil {
		return nil, err
	}
	if res != nil {
		return &model.TypedResourceContent{
			Type:     r.GetType(),
			Resource: res,
		}, nil
	}

	fs := memoryfs.New()
	_, _, err = download.DefaultRegistry.Download(nil, access, filepath.Join("/"), fs)
	if err != nil {
		return nil, err
	}

	typedResourceContent, err := h.Prepare(ctx, fs)
	if err != nil {
		return nil, err
	}

	_, err = h.cache.Put(ctx, r.GetCachingIdentity(ctx), typedResourceContent)
	if err != nil {
		return nil, err
	}

	return typedResourceContent, nil
}

func (h *BlueprintHandler) Prepare(ctx context.Context, fs vfs.FileSystem) (*model.TypedResourceContent, error) {
	bp, err := blueprint.BuildBlueprintFromPath(fs, "/")
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     mediatype.BlueprintType,
		Resource: bp,
	}, nil
}
