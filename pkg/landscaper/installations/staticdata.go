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

package installations

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils"
)

// GetDataFromSecretKeyRef fetches the referenced key from the cluster and parses in into a map.
func GetDataFromSecretKeyRef(ctx context.Context, client client.Client, ref *corev1.SecretKeySelector, namespace string) (map[string]interface{}, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, secret); err != nil {
		return nil, err
	}

	data, ok := secret.Data[ref.Key]
	if !ok {
		notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: secret.GroupVersionKind().Group, Resource: secret.GroupVersionKind().Kind}, secret.Name)
		return nil, errors.Wrapf(notFoundErr, "key %s not found in resource", ref.Key)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}

	return values, nil
}

// GetDataFromSecretKeyRef fetches the referenced key from the cluster and parses in into a map.
func GetDataFromSecretLabelSelectorRef(ctx context.Context, client client.Client, ref *lsv1alpha1.SecretLabelSelectorRef, namespace string) (map[string]interface{}, error) {
	secretList := &corev1.SecretList{}
	if err := client.List(ctx, secretList); err != nil {
		return nil, err
	}

	allValues := make(map[string]interface{})
	for _, secret := range secretList.Items {
		data, ok := secret.Data[ref.Key]
		if !ok {
			notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: secret.GroupVersionKind().Group, Resource: secret.GroupVersionKind().Kind}, secret.Name)
			return nil, errors.Wrapf(notFoundErr, "key %s not found in resource", ref.Key)
		}

		var values map[string]interface{}
		if err := yaml.Unmarshal(data, &values); err != nil {
			return nil, err
		}

		allValues = utils.MergeMaps(allValues, values)
	}

	return allValues, nil
}
