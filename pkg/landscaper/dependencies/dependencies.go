//// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////      http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//// See the License for the specific language governing permissions and
//// limitations under the License.
//
package dependencies

//
//import (
//	"errors"
//
//	"github.com/gardener/landscaper/pkg/landscaper/component"
//	"github.com/gardener/landscaper/pkg/landscaper/dataobject/jsonpath"
//
//	corev1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
//)
//
//// CheckImportSatisfaction traverses all component and checks if the imports of the current component are all satisfied
//func CheckImportSatisfaction(current *component.Component, components []*component.Component, landscapeConfig map[string]interface{}) error {
//	// todo: schrodit - parallelize execution and catch multierror
//	for _, importSpec := range current.Reference.Spec.Imports {
//
//		// check if the value can be found in the landscape config
//		if err := jsonpath.GetValue(importSpec.From, landscapeConfig, nil); err == nil {
//			// validate against type with openv3 value
//			continue
//		}
//
//		_, exportComponent, err := GetComponentForImport(importSpec, components)
//		if err != nil {
//			return err
//		}
//
//		//if exportSpec.Type != importSpec.Type {
//		//	return errors.New("export type has to be of the same type as the import")
//		//}
//
//		importStatus, ok := current.GetImportStatus(importSpec.From)
//		if !ok {
//			// if the state does not exist we assume that the component never ran
//			continue
//		}
//
//		// if the component already has a state for the import
//		// we have to check whether the generation has already changed
//
//		// todo: traverse through tree to root node
//		if importStatus.ConfigGeneration >= exportComponent.Reference.Status.ConfigGeneration {
//			return errors.New("config has not been updated yet")
//		}
//
//	}
//
//	return nil
//}
//
//func GetComponentForImport(importSpec corev1alpha1.DefinitionImportMapping, components []*component.Component) (*corev1alpha1.DefinitionExportMapping, *component.Component, error) {
//	for _, component := range components {
//		if component.Reference.Status.Phase != corev1alpha1.ComponentPhaseCompleted {
//			continue
//		}
//		for _, exportItem := range component.Reference.Spec.Exports {
//			if exportItem.To == importSpec.From {
//				return &exportItem, component, nil
//			}
//		}
//	}
//
//	return nil, nil, errors.New("no component found to satisfy the import")
//}
