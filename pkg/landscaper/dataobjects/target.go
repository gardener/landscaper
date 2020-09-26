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

package dataobjects

import (
	"encoding/json"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
)

// Target is the internal representation of a target.
type Target struct {
	Raw        *lsv1alpha1.Target
	FieldValue *lsv1alpha1.FieldValueDefinition
	Metadata   Metadata
}

// New creates a new internal dataobject.
func NewTarget() *Target {
	return &Target{}
}

// NewFromDataObject creates a new internal target instance from a raw target.
func NewFromTarget(target *lsv1alpha1.Target) (*Target, error) {
	return &Target{
		Raw:      target,
		Metadata: GetMetadataFromObject(target),
	}, nil
}

// GetData returns the target as internal go map.
func (t *Target) GetData() (interface{}, error) {
	raw, err := json.Marshal(t.Raw)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetContext sets the installation context for the given data object.
func (t *Target) SetContext(ctx string) *Target {
	t.Metadata.Context = ctx
	return t
}

// SetNamespace sets the namespace for the given data object.
func (t *Target) SetNamespace(ns string) *Target {
	t.Metadata.Namespace = ns
	return t
}

// SetSourceType sets the context for the given data object.
func (t *Target) SetSourceType(ctx lsv1alpha1.DataObjectSourceType) *Target {
	t.Metadata.SourceType = ctx
	return t
}

// SetSource sets the source for the given data object.
func (t *Target) SetSource(src string) *Target {
	t.Metadata.Source = src
	return t
}

// SetKey sets the key for the given data object.
func (t *Target) SetKey(key string) *Target {
	t.Metadata.Key = key
	return t
}

// Build creates a new data object based on the given data and metadata.
func (t Target) Build() (*lsv1alpha1.Target, error) {
	raw := &lsv1alpha1.Target{}
	if t.Raw != nil {
		raw = t.Raw
	}
	raw.Name = lsv1alpha1helper.GenerateDataObjectName(t.Metadata.Context, t.Metadata.Key)
	raw.Namespace = t.Metadata.Namespace
	SetMetadataFromObject(raw, t.Metadata)
	return raw, nil
}
