// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blueprintsregistry

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/spf13/afero"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Registry is the interface for the landscaper to get component definitions and their blob data.
type Registry interface {
	// GetBlueprint returns the blueprint for a resource of type "Blueprint"
	GetBlueprint(ctx context.Context, ref cdv2.Resource) (*v1alpha1.Blueprint, error)
	// GetBlob returns the blob content for a component definition.
	GetContent(ctx context.Context, ref cdv2.Resource) (afero.Fs, error)
}
