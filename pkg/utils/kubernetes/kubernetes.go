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

package kubernetes

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// CreateOrUpdate creates or updates the given object in the Kubernetes
// cluster. The object's desired state must be reconciled with the existing
// state inside the passed in callback MutateFn.
// It also correctly handles objects that have the generateName attribute set.
//
// The MutateFn is called regardless of creating or updating an object.
//
// It returns the executed operation and an error.
func CreateOrUpdate(ctx context.Context, c client.Client, obj runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	// check if the name key has to be generated
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	key := client.ObjectKey{Namespace: accessor.GetNamespace(), Name: accessor.GetName()}

	if accessor.GetName() == "" && accessor.GetGenerateName() != "" {
		if err := mutate(f, key, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := c.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	return controllerutil.CreateOrUpdate(ctx, c, obj, f)
}

// mutate wraps a MutateFn and applies validation to its result
func mutate(f controllerutil.MutateFn, key client.ObjectKey, obj runtime.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey, err := client.ObjectKeyFromObject(obj); err != nil || key != newKey {
		return fmt.Errorf("MutateFn cannot mutate object name and/or object namespace")
	}
	return nil
}

// GetStatusForContainer returns the container for a specific container
func GetStatusForContainer(containerStatus []corev1.ContainerStatus, name string) (corev1.ContainerStatus, error) {
	for _, status := range containerStatus {
		if status.Name == name {
			return status, nil
		}
	}
	return corev1.ContainerStatus{}, errors.New("container not found")
}

// OwnerOfGVK validates whether a instance of the given gvk is referenced
func OwnerOfGVK(ownerRefs []v1.OwnerReference, gvk schema.GroupVersionKind) (string, bool) {
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
func TypedObjectReferenceFromObject(obj runtime.Object, scheme *runtime.Scheme) (*v1alpha1.TypedObjectReference, error) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.TypedObjectReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		ObjectReference: v1alpha1.ObjectReference{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
		},
	}, nil
}

// HasFinalizer checks if the object constains a finalizer with the given name.
func HasFinalizer(obj v1.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}
