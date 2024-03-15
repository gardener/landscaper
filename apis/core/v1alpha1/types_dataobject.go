// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataObjectSourceType defines the context of a data object.
type DataObjectSourceType string

const (
	// ExportDataObjectSourceType is the data object type of a exported object.
	ExportDataObjectSourceType DataObjectSourceType = "export"
	// ExportDataObjectSourceType is the data object type of a imported object.
	ImportDataObjectSourceType DataObjectSourceType = "import"
)

// DataObjectTypeAnnotation defines the name of the annotation that specifies the type of the dataobject.
const DataObjectTypeAnnotation = "data.landscaper.gardener.cloud/type"

// DataObjectContextLabel defines the name of the label that specifies the context of the dataobject.
const DataObjectContextLabel = "data.landscaper.gardener.cloud/context"

// DataObjectSourceTypeLabel defines the name of the label that specifies the source type (import or export) of the dataobject.
const DataObjectSourceTypeLabel = "data.landscaper.gardener.cloud/sourceType"

// DataObjectKeyLabel defines the name of the label that specifies the export or imported key of the dataobject.
const DataObjectKeyLabel = "data.landscaper.gardener.cloud/key"

// DataObjectSourceLabel defines the name of the label that specifies the source of the dataobject.
const DataObjectSourceLabel = "data.landscaper.gardener.cloud/source"

// DataObjectIndexLabel defines the name of the label that specifies the index of the dataobject (for list-type imports)
const DataObjectIndexLabel = "data.landscaper.gardener.cloud/index"

// DataObjectTargetMapKeyLabel defines the label for the key in a target map.
const DataObjectTargetMapKeyLabel = "data.landscaper.gardener.cloud/targetmapkey"

// DataObjectJobIDLabel defines the job ID under which a data object was created.
const DataObjectJobIDLabel = "data.landscaper.gardener.cloud/jobid"

// DataObjectHashAnnotation defines the name of the annotation that specifies the hash of the data.
const DataObjectHashAnnotation = "data.landscaper.gardener.cloud/hash"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataObjectList contains a list of DataObject
type DataObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataObject `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=dobj;do
// +kubebuilder:printcolumn:name="Context",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/context']`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.metadata.labels['data\.landscaper\.gardener\.cloud\/key']`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DataObject are resources that can hold any kind json or yaml data.
type DataObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Data contains the data of the object as string.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Data AnyJSON `json:"data"`
}
