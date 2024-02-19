// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// This is the attempt to create a common interface for all imports to not always have big switch-case-statements when working with them.
// Unfortunately, the different imports behave quite differently, most notable are the differences between imports referring to a single object
// (single-type imports) and imports referring to multiple objects (list-type imports), which is why there are multiple methods which
// are only implemented for either type and calling them on a 'wrong' import will return a dummy value.
type ImportedBase interface {
	GetImportType() lsv1alpha1.ImportType
	// IsListTypeImport returns true if the import refers (or could refer) to multiple in-cluster objects.
	// List imports with zero or only one element are still considered list-type imports.
	// There are several methods which work for only either single-type or list-type imports, use this method to find out which ones to use.
	IsListTypeImport() bool
	// GetInClusterObject returns the in-cluster object referenced by this import
	GetInClusterObject() client.Object
	// GetInClusterObjects is the same as GetInClusterObject, but for list-type imports
	GetInClusterObjects() []client.Object
	// ComputeConfigGeneration computes the config generation for the import to compare it to the one stored in the import status
	GetListItems() []ImportedBase
	// GetImportReference returns the (non-hashed) name under which the imported object was exported. It is implemented for single-type imports only.
	GetImportReference() string
	// GetImportDefinition returns the import definition which caused this import.
	// Unfortunately, there is no common interface for import definitions, so it can only return interface{}.
	GetImportDefinition() interface{}
}

type Imported struct {
	ImportedBase
	importName string
}

func NewImported(importName string, data ImportedBase) *Imported {
	return &Imported{
		importName:   importName,
		ImportedBase: data,
	}
}

// GetImportName returns the name under which the import was imported.
func (imp *Imported) GetImportName() string {
	return imp.importName
}

// GetImportPath returns the field path for the import.
// This is helpful for providing better error messages.
func (imp *Imported) GetImportPath() *field.Path {
	fldPath := field.NewPath("spec").Child("imports")
	switch imp.GetImportType() {
	case lsv1alpha1.ImportTypeData:
		fldPath = fldPath.Child("data")
	case lsv1alpha1.ImportTypeTarget, lsv1alpha1.ImportTypeTargetList:
		fldPath = fldPath.Child("targets")
	}
	return fldPath.Child(imp.GetImportName())
}

// GetOwnerReference returns the owner reference.
// This is only meant for single-type imports, for list-type imports use GetOwnerReferences() instead.
func (imp *Imported) GetOwnerReference() *metav1.OwnerReference {
	if imp.IsListTypeImport() {
		return nil
	}
	obj := imp.GetInClusterObject()
	if obj == nil {
		return nil
	}
	return kutil.GetMainOwnerFromOwnerReferences(obj.GetOwnerReferences())
}

// GetOwnerReferences is the list-type import variant of GetOwnerReference.
// It returns a mapping from in-cluster object names to owner references.
// The result should contain one entry per item in the imported list.
func (imp *Imported) GetOwnerReferences() map[string]*metav1.OwnerReference {
	if !imp.IsListTypeImport() {
		return nil
	}
	objs := imp.GetInClusterObjects()
	if objs == nil {
		return nil
	}
	res := map[string]*metav1.OwnerReference{}
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		res[obj.GetName()] = kutil.GetMainOwnerFromOwnerReferences(obj.GetOwnerReferences())
	}
	return res
}

// GetImportReferences is the list-type import implementation for GetImportReference()
// It returns a mapping from in-cluster object names to import references.
func (imp *Imported) GetImportReferences() map[string]string {
	if !imp.IsListTypeImport() {
		return nil
	}
	res := map[string]string{}
	items := imp.GetListItems()
	for _, item := range items {
		obj := item.GetInClusterObject()
		if obj == nil {
			continue
		}
		res[obj.GetName()] = item.GetImportReference()
	}
	return res
}
