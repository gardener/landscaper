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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
)

// DataObject is the internal representation of a data object.
type DataObject struct {
	Raw  *lsv1alpha1.DataObject
	Data interface{}

	FieldValue *lsv1alpha1.DefinitionFieldValue
	Metadata   DataObjectMetadata
}

// DataObjectMetadata describes the metadata of a data object.
// This metadata is also represented as annotations/labels at the object.
type DataObjectMetadata struct {
	Namespace  string
	Context    string
	SourceType lsv1alpha1.DataObjectSourceType
	Source     string
	Key        string
}

// New creates a new internal dataobject.
func New() *DataObject {
	return &DataObject{}
}

// NewFromDataObject creates a new internal dataobject instance from a raw data object.
func NewFromDataObject(do *lsv1alpha1.DataObject) (*DataObject, error) {
	var data interface{}
	if err := yaml.Unmarshal(do.Data, &data); err != nil {
		return nil, err
	}
	return &DataObject{
		Raw:      do,
		Data:     data,
		Metadata: GetMetadataFromObject(do),
	}, nil
}

// GetMetadataFromObject read optional metadata from object's labels and annotations
func GetMetadataFromObject(objAcc metav1.Object) DataObjectMetadata {
	meta := DataObjectMetadata{}
	if objAcc.GetLabels() != nil {
		labels := objAcc.GetLabels()
		if context, ok := labels[lsv1alpha1.DataObjectContextLabel]; ok {
			meta.Context = context
		}
		if context, ok := labels[lsv1alpha1.DataObjectSourceTypeLabel]; ok {
			meta.SourceType = lsv1alpha1.DataObjectSourceType(context)
		}
		if src, ok := labels[lsv1alpha1.DataObjectSourceLabel]; ok {
			meta.Source = src
		}
		if key, ok := labels[lsv1alpha1.DataObjectKeyLabel]; ok {
			meta.Key = key
		}
	}
	return meta
}

// GetData searches its data for the given Javascript Object Notation path
// and unmarshals it into the given object
func (do *DataObject) GetData(path string, out interface{}) error {
	return jsonpath.GetValue(path, do.Data, out)
}

// SetData sets the data for the given object.
func (do *DataObject) SetData(data interface{}) *DataObject {
	do.Data = data
	return do
}

// SetContext sets the installation context for the given data object.
func (do *DataObject) SetContext(ctx string) *DataObject {
	do.Metadata.Context = ctx
	return do
}

// SetNamespace sets the namespace for the given data object.
func (do *DataObject) SetNamespace(ns string) *DataObject {
	do.Metadata.Namespace = ns
	return do
}

// SetSourceType sets the context for the given data object.
func (do *DataObject) SetSourceType(ctx lsv1alpha1.DataObjectSourceType) *DataObject {
	do.Metadata.SourceType = ctx
	return do
}

// SetSource sets the source for the given data object.
func (do *DataObject) SetSource(src string) *DataObject {
	do.Metadata.Source = src
	return do
}

// SetKey sets the key for the given data object.
func (do *DataObject) SetKey(key string) *DataObject {
	do.Metadata.Key = key
	return do
}

// Build creates a new data object based on the given data and metadata.
func (do DataObject) Build() (*lsv1alpha1.DataObject, error) {
	var (
		raw = &lsv1alpha1.DataObject{}
		err error
	)
	raw.Name = lsv1alpha1helper.GenerateDataObjectName(do.Metadata.Context, do.Metadata.Key)
	raw.Namespace = do.Metadata.Namespace
	raw.Data, err = yaml.Marshal(do.Data)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
