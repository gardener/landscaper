// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"context"
	"sync"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/pkg/components/model"
)

var Registry = New()

type ResourceHandler interface {
	GetResourceContent(ctx context.Context, r model.Resource, access ocm.ResourceAccess) (*model.TypedResourceContent, error)
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

	res, ok := r.handlers[typ]
	if !ok {
		return nil
	}
	return res
}
