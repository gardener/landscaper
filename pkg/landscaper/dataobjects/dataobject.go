// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
)

var _ ImportedBase = &DataObject{}

// DataObject is the internal representation of a data object.
type DataObject struct {
	Raw  *lsv1alpha1.DataObject
	Data interface{}

	FieldValue *lsv1alpha1.FieldValueDefinition
	Metadata   Metadata
	Def        *lsv1alpha1.DataImport
}

// Metadata describes the metadata of a data object.
// This metadata is also represented as annotations/labels at the object.
type Metadata struct {
	Namespace  string
	Context    string
	SourceType lsv1alpha1.DataObjectSourceType
	Source     string
	Key        string
	Hash       string
	Index      *int
}

// generateHash returns the internal data generation function for dataobjects or targets.
func generateHash(data []byte) string {
	h := sha1.New()
	_, _ = h.Write(data) // will never throw an error
	return hex.EncodeToString(h.Sum(nil))
}

// New creates a new internal dataobject.
func New() *DataObject {
	return &DataObject{}
}

// NewFromDataObject creates a new internal dataobject instance from a raw data object.
func NewFromDataObject(do *lsv1alpha1.DataObject) (*DataObject, error) {
	var data interface{}
	if err := yaml.Unmarshal(do.Data.RawMessage, &data); err != nil {
		return nil, err
	}
	return &DataObject{
		Raw:      do,
		Data:     data,
		Metadata: GetMetadataFromObject(do, do.Data.RawMessage),
	}, nil
}

// GetMetadataFromObject read optional metadata from object's labels and annotations
func GetMetadataFromObject(objAcc metav1.Object, data []byte) Metadata {
	meta := Metadata{}
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
		if rawIndex, ok := labels[lsv1alpha1.DataObjectIndexLabel]; ok {
			index, err := strconv.Atoi(rawIndex)
			if err == nil {
				meta.Index = pointer.Int(index)
			}
		}
	}
	if hash, ok := objAcc.GetAnnotations()[lsv1alpha1.DataObjectHashAnnotation]; ok {
		meta.Hash = hash
	} else if data != nil {
		// calculate the hash based on the data
		meta.Hash = generateHash(data)
	}
	return meta
}

// SetMetadataFromObject sets the given metadata as the object's labels and annotations
func SetMetadataFromObject(objAcc metav1.Object, meta Metadata) {
	labels := objAcc.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if len(meta.Context) != 0 {
		labels[lsv1alpha1.DataObjectContextLabel] = meta.Context
	} else {
		delete(labels, lsv1alpha1.DataObjectContextLabel)
	}
	if len(meta.SourceType) != 0 {
		labels[lsv1alpha1.DataObjectSourceTypeLabel] = string(meta.SourceType)
	} else {
		delete(labels, lsv1alpha1.DataObjectSourceTypeLabel)
	}
	if len(meta.Source) != 0 {
		labels[lsv1alpha1.DataObjectSourceLabel] = meta.Source
	} else {
		delete(labels, lsv1alpha1.DataObjectSourceLabel)
	}
	if len(meta.Key) != 0 {
		labels[lsv1alpha1.DataObjectKeyLabel] = meta.Key
	} else {
		delete(labels, lsv1alpha1.DataObjectKeyLabel)
	}
	if meta.Index != nil {
		labels[lsv1alpha1.DataObjectIndexLabel] = fmt.Sprint(*meta.Index)
	} else {
		delete(labels, lsv1alpha1.DataObjectIndexLabel)
	}

	objAcc.SetLabels(labels)

	ann := objAcc.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[lsv1alpha1.DataObjectHashAnnotation] = meta.Hash

	objAcc.SetAnnotations(ann)
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
	raw.Data.RawMessage, err = json.MarshalIndent(do.Data, "", "  ")
	if err != nil {
		return nil, err
	}
	do.Metadata.Hash = generateHash(raw.Data.RawMessage)
	SetMetadataFromObject(raw, do.Metadata)
	return raw, nil
}

// Apply applies data and metadata to a existing object.
func (do DataObject) Apply(raw *lsv1alpha1.DataObject) error {
	var (
		err error
	)
	raw.Name = lsv1alpha1helper.GenerateDataObjectName(do.Metadata.Context, do.Metadata.Key)
	raw.Namespace = do.Metadata.Namespace
	raw.Data.RawMessage, err = json.MarshalIndent(do.Data, "", "  ")
	if err != nil {
		return err
	}
	do.Metadata.Hash = generateHash(raw.Data.RawMessage)
	SetMetadataFromObject(raw, do.Metadata)
	return nil
}

// Imported interface

func (do *DataObject) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeData
}

func (do *DataObject) IsListTypeImport() bool {
	return false
}

func (do *DataObject) GetInClusterObject() client.Object {
	return do.Raw
}
func (do *DataObject) GetInClusterObjects() []client.Object {
	return nil
}

func (do *DataObject) ComputeConfigGeneration() string {
	if len(do.Metadata.Hash) != 0 {
		return do.Metadata.Hash
	}
	return generateHash(do.Raw.Data.RawMessage)
}

func (do *DataObject) GetListItems() []ImportedBase {
	return nil
}

func (do *DataObject) GetImportReference() string {
	return do.Def.DataRef
}

func (do *DataObject) GetImportDefinition() interface{} {
	return do.Def
}
