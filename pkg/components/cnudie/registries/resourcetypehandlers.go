package registries

import (
	"context"
	"io"
	"sync"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

var Registry = New()

type ResourceHandler interface {
	GetResourceContent(ctx context.Context, resource model.Resource, blobResolver model.BlobResolver) (*model.TypedResourceContent, error)
	Prepare(ctx context.Context, data io.Reader, info *types.BlobInfo) (*model.TypedResourceContent, error)
}

type ResourceHandlerRegistry struct {
	lock     sync.Mutex
	handlers map[string]ResourceHandler
}

func New() *ResourceHandlerRegistry {
	return &ResourceHandlerRegistry{
		handlers: map[string]ResourceHandler{},
	}
}

func (r *ResourceHandlerRegistry) Register(typ string, handler ResourceHandler) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.handlers[typ] = handler
}

func (r *ResourceHandlerRegistry) Get(typ string) ResourceHandler {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.handlers[typ]
}
