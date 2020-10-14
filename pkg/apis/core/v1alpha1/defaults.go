// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_DefinitionImport sets default values for the ImportDefinition objects
func SetDefaults_DefinitionImport(obj *ImportDefinition) {
	if obj.Required == nil {
		obj.Required = pointer.BoolPtr(true)
	}
}
