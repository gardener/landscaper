package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	k8sv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func BuildObjectFromTemplate(filePath string, settings map[string]any, obj client.Object) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	tmpl, err := template.New("tmpl").Parse(string(data))
	if err != nil {
		return err
	}

	var w bytes.Buffer
	if err := tmpl.Execute(&w, settings); err != nil {
		return err
	}

	if err := yaml.Unmarshal(w.Bytes(), obj); err != nil {
		return err
	}

	return nil
}

func CreateStateObjectFromTemplate(ctx context.Context, state *envtest.State, filePath string, settings map[string]any, obj client.Object) error {
	if err := BuildObjectFromTemplate(filePath, settings, obj); err != nil {
		return err
	}

	if err := state.Create(ctx, obj); err != nil {
		return err
	}

	return nil
}

func CreateClientObjectFromTemplate(ctx context.Context, cl client.Client, filePath string, settings map[string]any, obj client.Object) error {
	if err := BuildObjectFromTemplate(filePath, settings, obj); err != nil {
		return err
	}

	if err := cl.Create(ctx, obj); err != nil {
		return err
	}

	return nil
}

func CreateDataObjectFromFile(ctx context.Context, state *envtest.State, do *lsv1alpha1.DataObject, path string) error {
	if err := ReadResourceFromFile(do, path); err != nil {
		return err
	}
	do.SetNamespace(state.Namespace)
	if err := state.Create(ctx, do); err != nil {
		return err
	}
	return nil
}

func UpdateDataObjectFromFile(ctx context.Context, state *envtest.State, do *lsv1alpha1.DataObject, path string) error {
	doOld := &lsv1alpha1.DataObject{}
	if err := state.Client.Get(ctx, client.ObjectKeyFromObject(do), doOld); err != nil {
		return err
	}
	if err := ReadResourceFromFile(do, path); err != nil {
		return err
	}
	do.SetNamespace(state.Namespace)
	do.ResourceVersion = doOld.ResourceVersion
	return state.Client.Update(ctx, do)
}

func CreateNamespaceDataObjectFromFile(ctx context.Context, state *envtest.State, do *lsv1alpha1.DataObject, path string) error {
	if err := ReadResourceFromFile(do, path); err != nil {
		return err
	}
	do.SetNamespace(state.Namespace)
	SetDataObjectData(do, state.Namespace)
	if err := state.Create(ctx, do); err != nil {
		return err
	}
	return nil
}

func CreateContextFromFile(ctx context.Context, state *envtest.State, contxt *lsv1alpha1.Context, path string) error {
	if err := ReadResourceFromFile(contxt, path); err != nil {
		return err
	}
	contxt.SetNamespace(state.Namespace)
	if err := state.Create(ctx, contxt); err != nil {
		return err
	}
	return nil
}

func CreateInstallationFromFile(ctx context.Context, state *envtest.State, inst *lsv1alpha1.Installation, path string) error {
	if err := ReadResourceFromFile(inst, path); err != nil {
		return err
	}
	inst.Namespace = state.Namespace
	if err := state.Create(ctx, inst); err != nil {
		return err
	}
	return nil
}

func CreateServiceAccount(ctx context.Context, state *envtest.State, name, namespace string) (*k8sv1.ServiceAccount, error) {
	s := &k8sv1.ServiceAccount{}
	s.SetName(name)
	s.SetNamespace(namespace)
	if err := state.Client.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func DeleteServiceAccount(ctx context.Context, state *envtest.State, name, namespace string) error {
	s := &k8sv1.ServiceAccount{}
	s.SetName(name)
	s.SetNamespace(namespace)
	err := state.Client.Delete(ctx, s)
	return err
}

func UpdateInstallationFromFile(ctx context.Context, state *envtest.State, inst *lsv1alpha1.Installation, path string) error {
	instOld := &lsv1alpha1.Installation{}
	if err := state.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld); err != nil {
		return err
	}
	if err := ReadResourceFromFile(inst, path); err != nil {
		return err
	}
	inst.Namespace = state.Namespace
	inst.ResourceVersion = instOld.ResourceVersion
	return state.Update(ctx, inst)
}

func CheckConfigMap(ctx context.Context, state *envtest.State, name string, expectedData map[string]string) error {
	configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: name}
	configMap := &k8sv1.ConfigMap{}
	if err := state.Client.Get(ctx, configMapKey, configMap); err != nil {
		return err
	}
	return compareMaps(configMap.Data, expectedData)
}

func CheckDataObjectString(ctx context.Context, state *envtest.State, name string, expectedValue string) error {
	exportDo := &lsv1alpha1.DataObject{}
	exportDoKey := client.ObjectKey{Name: name, Namespace: state.Namespace}
	if err := state.Client.Get(ctx, exportDoKey, exportDo); err != nil {
		return err
	}
	actualValue := ""
	GetDataObjectData(exportDo, &actualValue)
	if actualValue != expectedValue {
		return fmt.Errorf("DataObject %s contains wrong value: actual %s, expected %s", name, actualValue, expectedValue)
	}
	return nil
}

func CheckDataObjectMap(ctx context.Context, state *envtest.State, name string, expectedData map[string]string) error {
	exportDo := &lsv1alpha1.DataObject{}
	exportDoKey := client.ObjectKey{Name: name, Namespace: state.Namespace}
	if err := state.Client.Get(ctx, exportDoKey, exportDo); err != nil {
		return err
	}
	actualData := map[string]string{}
	GetDataObjectData(exportDo, &actualData)
	return compareMaps(actualData, expectedData)
}

func compareMaps(actualData, expectedData map[string]string) error {
	if len(actualData) != len(expectedData) {
		return fmt.Errorf("map has %d entries, expected %d", len(actualData), len(expectedData))
	}
	for key, expectedValue := range expectedData {
		actualValue, ok := actualData[key]
		if !ok {
			return fmt.Errorf("map does not contain key %s", key)
		}
		if actualValue != expectedValue {
			return fmt.Errorf("map has wrong value for key %s", key)
		}
	}
	return nil
}
