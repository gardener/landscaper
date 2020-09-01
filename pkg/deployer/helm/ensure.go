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

package helm

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/pkg/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kubernetesutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string, exports map[string]interface{}) error {
	_, kubeClient, err := h.TargetClient()
	if err != nil {
		return err
	}

	objects := make([]*unstructured.Unstructured, 0)
	for name, content := range files {
		decodedObjects, err := h.decodeObjects(name, []byte(content))
		if err != nil {
			return err
		}
		// add possible export
		for _, obj := range decodedObjects {
			exports, err = h.addExport(exports, obj)
			if err != nil {
				return err
			}
		}

		objects = append(objects, decodedObjects...)
	}

	status := &helmv1alpha1.ProviderStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ProviderStatus",
		},
		ManagedResources: make([]lsv1alpha1.TypedObjectReference, len(objects)),
	}
	for i, obj := range objects {
		_, err = kubernetesutil.CreateOrUpdate(ctx, kubeClient, obj, func() error {
			return nil
		})

		if err != nil {
			return err
		}

		status.ManagedResources[i] = lsv1alpha1.TypedObjectReference{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
	}

	statusData, err := encodeStatus(status)
	if err != nil {
		return err
	}

	if err := h.createOrUpdateExport(ctx, exports); err != nil {
		return err
	}

	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	h.DeployItem.Status.ProviderStatus = statusData
	h.DeployItem.Status.ObservedGeneration = h.DeployItem.Generation
	return h.kubeClient.Status().Update(ctx, h.DeployItem)
}

func (h *Helm) createOrUpdateExport(ctx context.Context, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	secret.GenerateName = "mock-export-"
	secret.Namespace = h.DeployItem.Namespace
	if h.DeployItem.Status.ExportReference != nil {
		secret.Name = h.DeployItem.Status.ExportReference.Name
		secret.Namespace = h.DeployItem.Status.ExportReference.Namespace
	}

	_, err = kubernetesutil.CreateOrUpdate(ctx, h.kubeClient, secret, func() error {
		secret.Data = map[string][]byte{
			lsv1alpha1.DataObjectSecretDataKey: data,
		}
		return controllerutil.SetOwnerReference(h.DeployItem, secret, kubernetes.LandscaperScheme)
	})
	if err != nil {
		return err
	}

	h.DeployItem.Status.ExportReference = &lsv1alpha1.ObjectReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}

	return h.kubeClient.Status().Update(ctx, h.DeployItem)
}

func (h *Helm) DeleteFiles(ctx context.Context) error {
	status := &helmv1alpha1.ProviderStatus{}
	if _, _, err := serializer.NewCodecFactory(Helmscheme).UniversalDecoder().Decode(h.DeployItem.Status.ProviderStatus.Raw, nil, status); err != nil {
		return err
	}

	if len(status.ManagedResources) == 0 {
		controllerutil.RemoveFinalizer(&h.DeployItem.ObjectMeta, lsv1alpha1.LandscaperFinalizer)
		return h.kubeClient.Update(ctx, h.DeployItem)
	}

	_, kubeClient, err := h.TargetClient()
	if err != nil {
		return err
	}

	objects := make([]*unstructured.Unstructured, 0)
	for _, ref := range status.ManagedResources {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": ref.APIVersion,
				"kind":       ref.Kind,
				"metadata": map[string]interface{}{
					"name":      ref.Name,
					"namespace": ref.Namespace,
				},
			},
		}
		objects = append(objects, obj)
	}

	completed := true
	for _, obj := range objects {
		if err := kubeClient.Delete(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		completed = false
	}

	if !completed {
		return errors.New("not all items are deleted")
	}

	// remove finalizer
	controllerutil.RemoveFinalizer(&h.DeployItem.ObjectMeta, lsv1alpha1.LandscaperFinalizer)
	return h.kubeClient.Update(ctx, h.DeployItem)
}

func (h *Helm) decodeObjects(name string, data []byte) ([]*unstructured.Unstructured, error) {
	var (
		decoder    = yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024)
		decodedObj map[string]interface{}
		objects    = make([]*unstructured.Unstructured, 0)
	)

	for i := 0; true; i++ {
		if err := decoder.Decode(&decodedObj); err != nil {
			if err == io.EOF {
				break
			}
			h.log.Error(err, fmt.Sprintf("unable to decode resource %d of file %s", i, name))
			continue
		}

		if decodedObj == nil {
			continue
		}
		obj := &unstructured.Unstructured{Object: decodedObj}
		objects = append(objects, obj.DeepCopy())
	}
	return objects, nil
}

func (h *Helm) constructExportsFromValues(values map[string]interface{}) (map[string]interface{}, error) {
	exports := make(map[string]interface{})

	for _, export := range h.Configuration.ExportsFromManifests {
		if export.FromResource != nil {
			continue
		}

		var val interface{}
		if err := jsonpath.GetValue(export.JSONPath, values, &val); err != nil {
			return nil, err
		}

		newValue, err := jsonpath.Construct(export.Key, val)
		if err != nil {
			return nil, err
		}

		exports = utils.MergeMaps(exports, newValue)
	}

	return exports, nil
}

func (h *Helm) addExport(exports map[string]interface{}, obj *unstructured.Unstructured) (map[string]interface{}, error) {
	export := h.findResource(obj)
	if export == nil {
		return exports, nil
	}

	var val interface{}
	if err := jsonpath.GetValue(export.JSONPath, obj.Object, &val); err != nil {
		return nil, err
	}

	newValue, err := jsonpath.Construct(export.Key, val)
	if err != nil {
		return nil, err
	}

	return utils.MergeMaps(exports, newValue), nil
}

func (h *Helm) findResource(obj *unstructured.Unstructured) *helmv1alpha1.ExportFromManifestItem {
	for _, export := range h.Configuration.ExportsFromManifests {
		if export.FromResource == nil {
			continue
		}
		if export.FromResource.APIVersion != obj.GetAPIVersion() {
			continue
		}
		if export.FromResource.Kind != obj.GetKind() {
			continue
		}
		if export.FromResource.Name != obj.GetName() {
			continue
		}
		if export.FromResource.Namespace != obj.GetNamespace() {
			continue
		}
		return &export
	}
	return nil
}

func encodeStatus(status *helmv1alpha1.ProviderStatus) (runtime.RawExtension, error) {
	status.TypeMeta = metav1.TypeMeta{
		APIVersion: helmv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ProviderStatus",
	}

	raw := &runtime.RawExtension{}
	obj := status.DeepCopyObject()
	if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&obj, raw, nil); err != nil {
		return runtime.RawExtension{}, err
	}
	return *raw, nil
}
