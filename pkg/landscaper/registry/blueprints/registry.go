// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Registry is the interface for the landscaper to get component definitions and their blob data.
type Registry interface {
	// GetBlueprint returns the blueprint for a resource of type "Blueprint"
	GetBlueprint(ctx context.Context, ref cdv2.Resource) (*v1alpha1.Blueprint, error)
	// GetBlob returns the blob content for a component definition.
	GetContent(ctx context.Context, ref cdv2.Resource, fs vfs.FileSystem) error
}
