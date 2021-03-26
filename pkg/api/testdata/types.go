// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type SomeType struct {
	metav1.TypeMeta
	NumberString string
}

func (t *SomeType) DeepCopyObject() runtime.Object {
	return &SomeType{
		TypeMeta:     t.TypeMeta,
		NumberString: t.NumberString,
	}
}

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "somegroup.gardener.cloud", Version: runtime.APIVersionInternal}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Schema.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&SomeType{},
	)
	return nil
}
