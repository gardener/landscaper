package blueprint

import (
	"context"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/components/cache/blueprint"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/ocmfacade/registries"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

func init() {
	registries.Registry.Register(mediatype.BlueprintType, New())
	registries.Registry.Register(mediatype.OldBlueprintType, New())
}

type BlueprintHandler struct{}

func New() *BlueprintHandler {
	return &BlueprintHandler{}
}

func (h *BlueprintHandler) Prepare(ctx context.Context, fs vfs.FileSystem) (_ *model.TypedResourceContent, rerr error) {
	blueprint, err := blueprint.BuildBlueprintFromPath(fs, "/")
	if err != nil {
		return nil, err
	}

	return &model.TypedResourceContent{
		Type:     mediatype.BlueprintType,
		Resource: blueprint,
	}, nil
}
