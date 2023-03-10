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
	"github.com/gardener/landscaper/hack/testcluster/pkg/utils"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
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
	// if the default Landscaper test bin does not exist we default to the kubebuilder testenv default
	// that uses the KUBEBUILDER_ASSETS env var.
	if _, err := os.Stat(testBinPath); err == nil {
		if err := os.Setenv("TEST_ASSET_KUBE_APISERVER", filepath.Join(testBinPath, "kube-apiserver")); err != nil {
			return nil, err
		}
		if err := os.Setenv("TEST_ASSET_ETCD", filepath.Join(testBinPath, "etcd")); err != nil {
			return nil, err
		}
		if err := os.Setenv("TEST_ASSET_KUBECTL", filepath.Join(testBinPath, "kubectl")); err != nil {
			return nil, err
		}
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

	fakeClient, err := client.New(restConfig, client.Options{Scheme: api.LandscaperScheme})
	if err != nil {
		return nil, err
	}

	retryClient := NewRetryingClient(fakeClient, utils.NewDiscardLogger())
	e.Client = retryClient
	return retryClient, nil
}

// Stop stops the running dev environment
func (e *Environment) Stop() error {
	return e.Env.Stop()
}

// InitState creates a new isolated environment with its own namespace.
func (e *Environment) InitState(ctx context.Context) (*State, error) {
	return InitStateWithNamespace(ctx, e.Client, nil, false)
}

// InitStateWithNamespace creates a new isolated environment with its own namespace.
func InitStateWithNamespace(ctx context.Context, c client.Client, log utils.Logger, createSecondNamespace bool) (*State, error) {
	state := NewStateWithClient(log, c)

	// create a new testing namespace
	ns := &corev1.Namespace{}
	ns.GenerateName = "tests-"
	if err := c.Create(ctx, ns); err != nil {
		return nil, err
	}
	state.Namespace = ns.Name

	if createSecondNamespace {
		// create a second testing namespace
		ns2 := &corev1.Namespace{}
		ns2.GenerateName = "tests-"
		if err := c.Create(ctx, ns2); err != nil {
			return nil, err
		}
		state.Namespace2 = ns2.Name
	}

	return state, nil
}

// InitResources creates a new isolated environment with its own namespace.
func (e *Environment) InitResources(ctx context.Context, resourcesPath string) (*State, error) {
	return e.initResources(ctx, resourcesPath, false)
}

func (e *Environment) InitResourcesWithTwoNamespaces(ctx context.Context, resourcesPath string) (*State, error) {
	return e.initResources(ctx, resourcesPath, true)
}

// InitResources creates a new isolated environment with its own namespace.
func (e *Environment) initResources(ctx context.Context, resourcesPath string, createSecondNamespace bool) (*State, error) {
	state, err := InitStateWithNamespace(ctx, e.Client, nil, createSecondNamespace)
	if err != nil {
		return nil, err
	}

	if err := state.InitResourcesWithClient(ctx, e.Client, resourcesPath); err != nil {
		return nil, err
	}
	return state, err
}

// InitDefaultContextFromInst creates a default landsacpe context object from a installation.
func (e *Environment) InitDefaultContextFromInst(ctx context.Context, state *State, inst *lsv1alpha1.Installation) error {
	cdRef := installations.GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	lsCtx := &lsv1alpha1.Context{
		RepositoryContext: cdRef.RepositoryContext,
	}
	lsCtx.Name = lsv1alpha1.DefaultContextName
	lsCtx.Namespace = inst.Namespace
	return state.CreateWithClient(ctx, e.Client, lsCtx)
}

// CleanupState cleans up a test environment.
func (e *Environment) CleanupState(ctx context.Context, state *State) error {
	return state.CleanupStateWithClient(ctx, e.Client, WithCleanupTimeout(5*time.Second))
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

		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "unable to read file %s", path)
		}

		// template files
		tmpl, err := template.New("init").Funcs(templatingFunctions).Parse(string(data))
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer([]byte{})
		if err := tmpl.Execute(buf, map[string]string{"Namespace": state.Namespace, "Namespace2": state.Namespace2}); err != nil {
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
	decoder := serializer.NewCodecFactory(api.LandscaperScheme).UniversalDecoder()

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
	case TargetSyncGVK.Kind:
		targetSync := &lsv1alpha1.TargetSync{}
		if _, _, err := decoder.Decode(data, nil, targetSync); err != nil {
			return nil, fmt.Errorf("unable to decode file as target: %w", err)
		}
		state.TargetSyncs[types.NamespacedName{Name: targetSync.Name, Namespace: targetSync.Namespace}.String()] = targetSync
		return append(objects, targetSync), nil
	case ContextGVK.Kind:
		context := &lsv1alpha1.Context{}
		if _, _, err := decoder.Decode(data, nil, context); err != nil {
			return nil, fmt.Errorf("unable to decode file as context: %w", err)
		}
		return append(objects, context), nil
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
