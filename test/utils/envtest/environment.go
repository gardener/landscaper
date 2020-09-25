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

package envtest

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// Environment is a internal landcaper test environment
type Environment struct {
	Env *envtest.Environment

	Client client.Client
}

// New creates a new test environment with the landscaper known crds.
func New(projectRoot string) (*Environment, error) {
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}
	testBinPath := filepath.Join(projectRoot, "tmp", "test", "bin")
	if err := os.Setenv("TEST_ASSET_KUBE_APISERVER", filepath.Join(testBinPath, "kube-apiserver")); err != nil {
		return nil, err
	}
	if err := os.Setenv("TEST_ASSET_ETCD", filepath.Join(testBinPath, "etcd")); err != nil {
		return nil, err
	}
	if err := os.Setenv("TEST_ASSET_KUBECTL", filepath.Join(testBinPath, "kubectl")); err != nil {
		return nil, err
	}
	return &Environment{
		Env: &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join(projectRoot, "crd")},
		},
	}, nil
}

// Start starts the fake environment and creates a client for the started kubernetes cluster.
func (e *Environment) Start() (client.Client, error) {
	restConfig, err := e.Env.Start()
	if err != nil {
		return nil, err
	}

	fakeClient, err := client.New(restConfig, client.Options{Scheme: kubernetes.LandscaperScheme})
	if err != nil {
		return nil, err
	}

	e.Client = fakeClient
	return fakeClient, nil
}

// Stop stops the running dev environment
func (e *Environment) Stop() error {
	return e.Env.Stop()
}

// InitResources creates a new isolated environment with its own namespace.
func (e *Environment) InitResources(ctx context.Context, resourcesPath string) (*State, error) {
	state := &State{
		Installations: make(map[string]*lsv1alpha1.Installation),
		Executions:    make(map[string]*lsv1alpha1.Execution),
		DeployItems:   make(map[string]*lsv1alpha1.DeployItem),
		DataObjects:   make(map[string]*lsv1alpha1.DataObject),
		Secrets:       make(map[string]*corev1.Secret),
	}
	// create a new testing namespace
	ns := &corev1.Namespace{}
	ns.GenerateName = "unit-tests-"
	if err := e.Client.Create(ctx, ns); err != nil {
		return nil, err
	}
	state.Namespace = ns.Name

	// parse state and create resources in cluster
	resources, err := parseResources(resourcesPath, state)
	if err != nil {
		return nil, err
	}

	for _, obj := range resources {
		if err := e.Client.Create(ctx, obj); err != nil {
			return nil, err
		}
		if err := e.Client.Status().Update(ctx, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
	}

	return state, nil
}

// CleanupState cleans up a test environment.
// todo: remove finalizers iof all objects in state
func (e *Environment) CleanupState(ctx context.Context, state *State) error {
	for _, obj := range state.DeployItems {
		if err := e.Client.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := e.removeFinalizer(ctx, obj); err != nil {
			return err
		}
		if err := e.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	for _, obj := range state.Executions {
		if err := e.Client.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := e.removeFinalizer(ctx, obj); err != nil {
			return err
		}
		if err := e.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	for _, obj := range state.Installations {
		if err := e.Client.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := e.removeFinalizer(ctx, obj); err != nil {
			return err
		}
		if err := e.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	for _, obj := range state.Secrets {
		if err := e.Client.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if err := e.removeFinalizer(ctx, obj); err != nil {
			return err
		}
		if err := e.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	ns := &corev1.Namespace{}
	ns.Name = state.Namespace
	return e.Client.Delete(ctx, ns)
}

func (e *Environment) removeFinalizer(ctx context.Context, object metav1.Object) error {
	if len(object.GetFinalizers()) == 0 {
		return nil
	}

	object.SetFinalizers([]string{})
	return e.Client.Update(ctx, object.(runtime.Object))
}

func parseResources(path string, state *State) ([]runtime.Object, error) {
	objects := make([]runtime.Object, 0)
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %s", path)
		}

		// template files
		tmpl, err := template.New("init").Funcs(templatingFunctions).Parse(string(data))
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer([]byte{})
		if err := tmpl.Execute(buf, map[string]string{"Namespace": state.Namespace}); err != nil {
			return err
		}

		var (
			decoder    = yaml.NewYAMLOrJSONDecoder(buf, 1024)
			decodedObj json.RawMessage
		)

		for {
			if err := decoder.Decode(&decodedObj); err != nil {
				if err == io.EOF {
					return nil
				}
				continue
			}

			objects, err = decodeAndAppendLSObject(decodedObj, objects, state)
			if err != nil {
				return errors.Wrapf(err, "unable to decode file %s", path)
			}

		}
	})
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func decodeAndAppendLSObject(data []byte, objects []runtime.Object, state *State) ([]runtime.Object, error) {
	var allErrors *multierror.Error
	decoder := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder()

	inst := &lsv1alpha1.Installation{}
	_, _, err := decoder.Decode(data, nil, inst)
	if err == nil {
		state.Installations[types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace}.String()] = inst
		return append(objects, inst), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	exec := &lsv1alpha1.Execution{}
	_, _, err = decoder.Decode(data, nil, exec)
	if err == nil {
		state.Executions[types.NamespacedName{Name: exec.Name, Namespace: exec.Namespace}.String()] = exec
		return append(objects, exec), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	deployItem := &lsv1alpha1.DeployItem{}
	_, _, err = decoder.Decode(data, nil, deployItem)
	if err == nil {
		state.DeployItems[types.NamespacedName{Name: deployItem.Name, Namespace: deployItem.Namespace}.String()] = deployItem
		return append(objects, deployItem), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	dataObject := &lsv1alpha1.DataObject{}
	_, _, err = decoder.Decode(data, nil, dataObject)
	if err == nil {
		state.DataObjects[types.NamespacedName{Name: dataObject.Name, Namespace: dataObject.Namespace}.String()] = dataObject
		return append(objects, dataObject), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	secret := &corev1.Secret{}
	_, _, err = decoder.Decode(data, nil, secret)
	if err == nil {

		// add stringdata and data
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		for key, data := range secret.StringData {
			secret.Data[key] = []byte(data)
		}

		state.Secrets[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}.String()] = secret
		return append(objects, secret), nil
	}
	allErrors = multierror.Append(allErrors, errors.Wrap(err, "unable to decode file"))

	return nil, allErrors
}

var templatingFunctions = template.FuncMap{
	"dataObjectContext": func(namespace, name string) string {
		return lsv1alpha1helper.DataObjectSourceFromInstallation(&lsv1alpha1.Installation{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		})
	},
	"executionDataObjectNameExec": func(namespace, name string) string {
		return lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromExecution(&lsv1alpha1.Execution{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}), "")
	},
	"dataObjectName": func(context, name string) string {
		return lsv1alpha1helper.GenerateDataObjectName(context, name)
	},
}
