package registries

import (
	"context"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/pkg/components/model"
)

var Registry = New()

type ResourceHandler interface {
	Prepare(ctx context.Context, fs vfs.FileSystem) (*model.TypedResourceContent, error)
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
