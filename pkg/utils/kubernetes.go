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

package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// OwnerOfGVK validates whether a instance of the given gvk is referenced
func OwnerOfGVK(ownerRefs []metav1.OwnerReference, gvk schema.GroupVersionKind) (string, bool) {
	for _, ownerRef := range ownerRefs {
		gv, err := schema.ParseGroupVersion(ownerRef.APIVersion)
		if err != nil {
			continue
		}
		if gv.Group == gvk.Group && ownerRef.Kind == gvk.Kind {
			return ownerRef.Name, true
		}
	}
	return "", false
}

// TypedObjectReferenceFromObject creates a typed object reference from a object.
func TypedObjectReferenceFromObject(obj runtime.Object, scheme *runtime.Scheme) (*lsv1alpha1.TypedObjectReference, error) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, err
	}

	return &lsv1alpha1.TypedObjectReference{
		APIGroup: gvk.GroupVersion().String(),
		Kind:     gvk.Kind,
		ObjectReference: lsv1alpha1.ObjectReference{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
		},
	}, nil
}

// HasFinalizer checks if the object constains a finalizer with the given name.
func HasFinalizer(obj metav1.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}
