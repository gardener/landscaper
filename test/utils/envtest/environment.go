// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// Environment is a internal landscaper test environment
type Environment struct {
	Env    *envtest.Environment
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
			CRDDirectoryPaths: []string{filepath.Join(projectRoot, ".crd")},
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

// InitState creates a new isolated environment with its own namespace.
func (e *Environment) InitState(ctx context.Context) (*State, error) {
	return InitStateWithNamespace(ctx, e.Client)
}

// InitState creates a new isolated environment with its own namespace.
func InitStateWithNamespace(ctx context.Context, c client.Client) (*State, error) {
	state := NewState()
	// create a new testing namespace
	ns := &corev1.Namespace{}
	ns.GenerateName = "tests-"
	if err := c.Create(ctx, ns); err != nil {
		return nil, err
	}
	state.Namespace = ns.Name

	return state, nil
}

// InitResources creates a new isolated environment with its own namespace.
func (e *Environment) InitResources(ctx context.Context, resourcesPath string) (*State, error) {
	state, err := e.InitState(ctx)
	if err != nil {
		return nil, err
	}

	if err := state.InitResources(ctx, e.Client, resourcesPath); err != nil {
		return nil, err
	}
	return state, err
}

// CleanupState cleans up a test environment.
func (e *Environment) CleanupState(ctx context.Context, state *State) error {
	t := 5 * time.Second
	return state.CleanupState(ctx, e.Client, &t)
}

func parseResources(path string, state *State) ([]client.Object, error) {
	objects := make([]client.Object, 0)
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

func decodeAndAppendLSObject(data []byte, objects []client.Object, state *State) ([]client.Object, error) {
	decoder := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder()

	_, gvk, err := decoder.Decode(data, nil, &unstructured.Unstructured{})
	if err != nil {
		return nil, fmt.Errorf("unable to decode object into unstructured: %w", err)
	}

	switch gvk.Kind {
	case InstallationGVK.Kind:
		inst := &lsv1alpha1.Installation{}
		if _, _, err := decoder.Decode(data, nil, inst); err != nil {
			return nil, fmt.Errorf("unable to decode file as installation: %w", err)
		}
		state.Installations[types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace}.String()] = inst
		return append(objects, inst), nil
	case ExecutionGVK.Kind:
		exec := &lsv1alpha1.Execution{}
		if _, _, err := decoder.Decode(data, nil, exec); err != nil {
			return nil, fmt.Errorf("unable to decode file as execution: %w", err)
		}
		state.Executions[types.NamespacedName{Name: exec.Name, Namespace: exec.Namespace}.String()] = exec
		return append(objects, exec), nil
	case DeployItemGVK.Kind:
		deployItem := &lsv1alpha1.DeployItem{}
		if _, _, err := decoder.Decode(data, nil, deployItem); err != nil {
			return nil, fmt.Errorf("unable to decode file as deploy item: %w", err)
		}
		state.DeployItems[types.NamespacedName{Name: deployItem.Name, Namespace: deployItem.Namespace}.String()] = deployItem
		return append(objects, deployItem), nil
	case DataObjectGVK.Kind:
		dataObject := &lsv1alpha1.DataObject{}
		if _, _, err := decoder.Decode(data, nil, dataObject); err != nil {
			return nil, fmt.Errorf("unable to decode file as data object: %w", err)
		}
		state.DataObjects[types.NamespacedName{Name: dataObject.Name, Namespace: dataObject.Namespace}.String()] = dataObject
		return append(objects, dataObject), nil
	case TargetGVK.Kind:
		target := &lsv1alpha1.Target{}
		if _, _, err := decoder.Decode(data, nil, target); err != nil {
			return nil, fmt.Errorf("unable to decode file as target: %w", err)
		}
		state.Targets[types.NamespacedName{Name: target.Name, Namespace: target.Namespace}.String()] = target
		return append(objects, target), nil
	case SecretGVK.Kind:
		secret := &corev1.Secret{}
		if _, _, err := decoder.Decode(data, nil, secret); err != nil {
			return nil, fmt.Errorf("unable to decode file as secret: %w", err)
		}
		// add stringdata and data
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		for key, data := range secret.StringData {
			secret.Data[key] = []byte(data)
		}

		state.Secrets[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}.String()] = secret
		return append(objects, secret), nil
	case ConfigMapGVK.Kind:
		cm := &corev1.ConfigMap{}
		if _, _, err := decoder.Decode(data, nil, cm); err != nil {
			return nil, fmt.Errorf("unable to decode file as configmap: %w", err)
		}
		state.ConfigMaps[types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}.String()] = cm
		return append(objects, cm), nil
	default:
		return objects, nil
	}
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
		n := lsv1alpha1helper.GenerateDataObjectName(context, name)
		return n
	},
}
