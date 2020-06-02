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
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// importsAreSatisfied traverses through all components and validates if all imports are
// satisfied with the correct version
func (a *actuator) importsAreSatisfied(ctx context.Context, landscapeConfig map[string]interface{}, current *v1alpha1.ComponentInstallation) error {
	//internalComponent, err := component.New(current)
	//if err != nil {
	//	return err
	//}
	//
	//components := &v1alpha1.ComponentInstallationList{}
	//if err := a.c.List(ctx, components); err != nil {
	//	return errors.Wrap(err, "unable to list components")
	//}
	//
	//internalComponents := make([]*component.Component, len(components.Items))
	//for i, c := range components.Items {
	//	internalComponents[i], err = component.New(c.DeepCopy())
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	//return dependencies.CheckImportSatisfaction(internalComponent, internalComponents, landscapeConfig)
	return nil
}

// getImports traverses through all components and
// collects and merges the imports
func (a *actuator) collectImports(ctx context.Context, landscapeConfig map[string]interface{}, current *v1alpha1.ComponentInstallation) (map[string]interface{}, error) {
	//var err error
	//
	//components := &v1alpha1.ComponentInstallationList{}
	//if err := a.c.List(ctx, components); err != nil {
	//	return nil, errors.Wrap(err, "unable to list components")
	//}
	//internalComponents := make([]*component.Component, len(components.Items))
	//for i, c := range components.Items {
	//	internalComponents[i], err = component.New(c.DeepCopy())
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//importConfig := make(map[string]interface{})
	//for _, importSpec := range current.Spec.Imports {
	//
	//	var val interface{}
	//	if err := jsonpath.GetValue(importSpec.From, landscapeConfig, &val); err == nil {
	//		exportValue, err := jsonpath.Construct(importSpec.To, val)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		importConfig = utils.MergeMaps(importConfig, exportValue)
	//		continue
	//	}
	//
	//	exportSpec, exportComponent, err := dependencies.GetComponentForImport(importSpec, internalComponents)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	// need to get all deployitems to get the export data
	//	exportedConfig := make(map[string]interface{})
	//	for _, executionState := range exportComponent.Info.Status.DeployItemReferences {
	//		deployItem := &v1alpha1.DeployItem{}
	//		if err := a.c.Get(ctx, client.ObjectKey{Name: executionState.Resource.Name, Namespace: executionState.Resource.Namespace}, deployItem); err != nil {
	//			return nil, err
	//		}
	//
	//		export, err := a.GetExportValueFromDeployItem(ctx, deployItem)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		exportedConfig = utils.MergeMaps(exportedConfig, export)
	//	}
	//
	//	var value interface{}
	//	if err := jsonpath.GetValue(exportSpec.From, exportedConfig, &value); err != nil {
	//		return nil, err
	//	}
	//
	//	exportValue, err := jsonpath.Construct(importSpec.To, value)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	importConfig = utils.MergeMaps(importConfig, exportValue)
	//}
	//
	//return importConfig, nil
	return nil, nil
}

func (a *actuator) GetExportValueFromDeployItem(ctx context.Context, deployItem *v1alpha1.DeployItem) (map[string]interface{}, error) {
	//if deployItem.Status.Export == nil {
	//	return make(map[string]interface{}), nil
	//}
	//
	//if len(deployItem.Status.Export.Value) != 0 {
	//	data := make(map[string]interface{})
	//	if err := json.Unmarshal([]byte(deployItem.Status.Export.Value), &data); err != nil {
	//		return nil, err
	//	}
	//	return data, nil
	//}
	//
	//if deployItem.Status.Export.ValueRef == nil {
	//	return nil, errors.New("no export value provided")
	//}
	//
	//secretRef := deployItem.Status.Export.ValueRef.SecretRef
	//secret := &corev1.Secret{}
	//if err := a.c.Get(ctx, client.ObjectKey{Name: secretRef.Name, Namespace: deployItem.Namespace}, secret); err != nil {
	//	return nil, err
	//}
	//
	//value, ok := secret.Data[secretRef.Key]
	//if !ok {
	//	return nil, fmt.Errorf("no data for key %s in secret %s", secretRef.Key, secretRef.Name)
	//}
	//
	//data := make(map[string]interface{})
	//if err := json.Unmarshal(value, &data); err != nil {
	//	return nil, err
	//}
	//return data, nil
	return nil, nil
}
