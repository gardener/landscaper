// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
)

func newTestResource(res *types.Resource) *TestResource {
	return &TestResource{
		resource: res,
	}
}

type TestResource struct {
	resource *types.Resource
}

var _ model.Resource = &TestResource{}

func (r *TestResource) GetName() string {
	return r.resource.GetName()
}

func (r *TestResource) GetVersion() string {
	return r.resource.GetVersion()
}

func (r *TestResource) GetType() string {
	return r.resource.GetType()
}

func (r *TestResource) GetAccessType() string {
	return r.resource.Access.GetType()
}

func (r *TestResource) GetResource() (*types.Resource, error) {
	return r.resource, nil
}

func (r *TestResource) GetTypedContent(ctx context.Context) (*model.TypedResourceContent, error) {
	return nil, fmt.Errorf("no handler found for resource type %s", r.GetType())
}

func (r *TestResource) GetCachingIdentity(ctx context.Context) string {
	return ""
}
