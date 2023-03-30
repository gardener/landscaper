// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package operation

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model"
)

// Builder implements the builder-pattern to craft the operation
type Builder struct {
	client            client.Client
	scheme            *runtime.Scheme
	eventRecorder     record.EventRecorder
	componentRegistry model.RegistryAccess
}

// NewBuilder creates a new operation builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Client sets the kubernetes client.
func (b *Builder) Client(c client.Client) *Builder {
	b.client = c
	return b
}

// Scheme sets the kubernetes scheme.
func (b *Builder) Scheme(s *runtime.Scheme) *Builder {
	b.scheme = s
	return b
}

// ComponentRegistry sets the component registry.
func (b *Builder) ComponentRegistry(registry model.RegistryAccess) *Builder {
	b.componentRegistry = registry
	return b
}

// WithEventRecorder sets a event recorder.
func (b *Builder) WithEventRecorder(er record.EventRecorder) *Builder {
	b.eventRecorder = er
	return b
}

func (b *Builder) applyDefaults(ctx context.Context) {
	if b.scheme == nil {
		b.scheme = api.LandscaperScheme
	}
	if b.eventRecorder == nil {
		b.eventRecorder = record.NewFakeRecorder(1024)
	}
}

func (b *Builder) validate() error {
	if b.client == nil {
		return errors.New("a kubernetes client must be set")
	}
	if b.componentRegistry == nil {
		return errors.New("a component registry must be set")
	}
	return nil
}

// Build creates a new operation.
func (b *Builder) Build(ctx context.Context) (*Operation, error) {
	b.applyDefaults(ctx)
	if err := b.validate(); err != nil {
		return nil, err
	}

	return &Operation{
		client:            b.client,
		scheme:            b.scheme,
		eventRecorder:     b.eventRecorder,
		componentRegistry: b.componentRegistry,
	}, nil
}
