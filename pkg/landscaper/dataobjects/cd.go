// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ ImportedBase = &ComponentDescriptor{}

type CDReferenceType string

const RegistryReference = CDReferenceType("registry")
const SecretReference = CDReferenceType("secret")
const ConfigMapReference = CDReferenceType("configmap")
const DataReference = CDReferenceType("data")

// ComponentDescriptor is the internal representation of a component descriptor.
type ComponentDescriptor struct {
	RefType      CDReferenceType
	RegistryRef  *lsv1alpha1.ComponentDescriptorReference
	SecretRef    *lsv1alpha1.SecretReference
	ConfigMapRef *lsv1alpha1.ConfigMapReference
	Descriptor   *cdv2.ComponentDescriptor
	Owner        *metav1.OwnerReference
	Def          *lsv1alpha1.ComponentDescriptorImport
}

// NewComponentDescriptor creates a new internal component descriptor.
func NewComponentDescriptor() *ComponentDescriptor {
	return &ComponentDescriptor{}
}

// GetData returns the component descriptor as internal go map.
func (cd *ComponentDescriptor) GetData() (interface{}, error) {
	raw, err := json.Marshal(cd.Descriptor)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetOwner sets the owner for the given data object.
func (cd *ComponentDescriptor) SetOwner(own *metav1.OwnerReference) *ComponentDescriptor {
	cd.Owner = own
	return cd
}

// SetRegistryReference sets the component descriptor reference for the given data object.
func (cd *ComponentDescriptor) SetRegistryReference(rr *lsv1alpha1.ComponentDescriptorReference) *ComponentDescriptor {
	cd.RegistryRef = rr
	cd.RefType = RegistryReference
	return cd
}

// SetSecretReference sets the component descriptor reference for the given data object.
func (cd *ComponentDescriptor) SetSecretReference(sr *lsv1alpha1.SecretReference) *ComponentDescriptor {
	cd.SecretRef = sr
	cd.RefType = SecretReference
	return cd
}

// SetConfigMapReference sets the component descriptor reference for the given data object.
func (cd *ComponentDescriptor) SetConfigMapReference(cmr *lsv1alpha1.ConfigMapReference) *ComponentDescriptor {
	cd.ConfigMapRef = cmr
	cd.RefType = ConfigMapReference
	return cd
}

// SetDescriptor sets the component descriptor data for the given data object.
func (cd *ComponentDescriptor) SetDescriptor(raw *cdv2.ComponentDescriptor) *ComponentDescriptor {
	cd.Descriptor = raw
	return cd
}

// Imported interface

func (cd *ComponentDescriptor) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeComponentDescriptor
}

func (cd *ComponentDescriptor) IsListTypeImport() bool {
	return false
}

func (cd *ComponentDescriptor) GetInClusterObject() client.Object {
	// component descriptors are not represented as in-cluster landscaper objects
	return nil
}
func (cd *ComponentDescriptor) GetInClusterObjects() []client.Object {
	return nil
}

func (cd *ComponentDescriptor) ComputeConfigGeneration() string {
	return ""
}

func (cd *ComponentDescriptor) GetListItems() []ImportedBase {
	return nil
}

func (cd *ComponentDescriptor) GetImportReference() string {
	// component descriptors cannot be exported
	// and references to parent imports are resolved and replaced during subinstallation rendering
	return ""
}

func (cd *ComponentDescriptor) GetImportDefinition() interface{} {
	return cd.Def
}
