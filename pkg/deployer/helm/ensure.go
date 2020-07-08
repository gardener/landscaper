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
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	kubernetesutil "github.com/gardener/landscaper/test/utils/kubernetes"
)

func (h *Helm) ApplyFiles(ctx context.Context, files map[string]string) error {
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
		objects = append(objects, decodedObjects...)
	}

	status := &Status{
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
			APIGroup: obj.GetAPIVersion(),
			Kind:     obj.GetKind(),
			ObjectReference: lsv1alpha1.ObjectReference{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
	}

	statusData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	h.DeployItem.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
	h.DeployItem.Status.ProviderStatus = statusData
	h.DeployItem.Status.ObservedGeneration = h.DeployItem.Generation
	return h.kubeClient.Status().Update(ctx, h.DeployItem)
}

func (h *Helm) DeleteFiles(ctx context.Context, files map[string]string) error {
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
		objects = append(objects, decodedObjects...)
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
