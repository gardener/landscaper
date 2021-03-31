// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gardener/landscaper/pkg/api/testdata"
)

type SomeType struct {
	metav1.TypeMeta
	Number int `json:"number"`
}

func (t *SomeType) DeepCopyObject() runtime.Object {
	return &SomeType{
		TypeMeta: t.TypeMeta,
		Number:   t.Number,
	}
}

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "somegroup.gardener.cloud", Version: "v2"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addConversionFuncs)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Schema.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&SomeType{},
	)
	return nil
}

func addConversionFuncs(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*SomeType)(nil), (*testdata.SomeType)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v2_AnyJSON_To_testdata_AnyJSON(a.(*SomeType), b.(*testdata.SomeType), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*testdata.SomeType)(nil), (*SomeType)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_testdata_AnyJSON_To_v2_AnyJSON(a.(*testdata.SomeType), b.(*SomeType), scope)
	}); err != nil {
		return err
	}
	return nil
}

func Convert_v2_AnyJSON_To_testdata_AnyJSON(a *SomeType, b *testdata.SomeType, scope conversion.Scope) error {
	b.NumberString = strconv.Itoa(a.Number)
	return nil
}

func Convert_testdata_AnyJSON_To_v2_AnyJSON(a *testdata.SomeType, b *SomeType, scope conversion.Scope) error {
	var err error
	b.Number, err = strconv.Atoi(a.NumberString)
	if err != nil {
		return err
	}
	return nil
}
