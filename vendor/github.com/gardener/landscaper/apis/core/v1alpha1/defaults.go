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

// SetDefaults_Blueprint sets default values for blueprint objects
func SetDefaults_Blueprint(obj *Blueprint) {
	if len(obj.JSONSchemaVersion) == 0 {
		obj.JSONSchemaVersion = "https://json-schema.org/draft/2019-09/schema"
	}

	SetDefaults_DefinitionImport(&obj.Imports)
	SetDefaults_DefinitionExport(&obj.Exports)
}

// SetDefaults_DefinitionImport sets default values for the ImportDefinition objects
func SetDefaults_DefinitionImport(imports *ImportDefinitionList) {
	if imports == nil {
		return
	}
	for i := 0; i < len(*imports); i++ {
		imp := &(*imports)[i]
		if imp.Required == nil {
			imp.Required = pointer.BoolPtr(true)
		}
		SetDefaults_DefinitionImport(&imp.ConditionalImports)
		if len(imp.Type) != 0 {
			// type is already set
			continue
		}
		if imp.Schema != nil && len(imp.TargetType) != 0 {
			// definition is invalid
			continue
		}
		if imp.Schema != nil {
			imp.Type = ImportTypeData
		} else if len(imp.TargetType) != 0 {
			imp.Type = ImportTypeTarget
		}
	}
}

// SetDefaults_DefinitionExport sets default values for the ImportDefinition objects
func SetDefaults_DefinitionExport(exports *ExportDefinitionList) {
	if exports == nil {
		return
	}
	for i := 0; i < len(*exports); i++ {
		exp := &(*exports)[i]
		if len(exp.Type) != 0 {
			// type is already set
			continue
		}
		if exp.Schema != nil && len(exp.TargetType) != 0 {
			// definition is invalid
			continue
		}
		if exp.Schema != nil {
			exp.Type = ExportTypeData
		} else if len(exp.TargetType) != 0 {
			exp.Type = ExportTypeTarget
		}
	}
}

// SetDefaults_Installation sets default values for installation objects
func SetDefaults_Installation(obj *Installation) {

	// default the repository context to "default"
	if len(obj.Spec.Context) == 0 {
		obj.Spec.Context = DefaultContextName
	}

	// default the namespace of imports
	for i, dataImport := range obj.Spec.Imports.Data {
		if dataImport.ConfigMapRef != nil {
			if len(dataImport.ConfigMapRef.Namespace) == 0 {
				obj.Spec.Imports.Data[i].ConfigMapRef.Namespace = obj.GetNamespace()
			}
		}
		if dataImport.SecretRef != nil {
			if len(dataImport.SecretRef.Namespace) == 0 {
				obj.Spec.Imports.Data[i].SecretRef.Namespace = obj.GetNamespace()
			}
		}
	}
	for i, cdImport := range obj.Spec.Imports.ComponentDescriptors {
		if cdImport.ConfigMapRef != nil {
			if len(cdImport.ConfigMapRef.Namespace) == 0 {
				obj.Spec.Imports.ComponentDescriptors[i].ConfigMapRef.Namespace = obj.GetNamespace()
			}
		}
		if cdImport.SecretRef != nil {
			if len(cdImport.SecretRef.Namespace) == 0 {
				obj.Spec.Imports.ComponentDescriptors[i].SecretRef.Namespace = obj.GetNamespace()
			}
		}
		if cdImport.Ref != nil {
			if cdImport.Ref.RepositoryContext == nil {
				if obj.Spec.ComponentDescriptor != nil && obj.Spec.ComponentDescriptor.Reference != nil {
					cdImport.Ref.RepositoryContext = obj.Spec.ComponentDescriptor.Reference.RepositoryContext
				}
			}
		}

		if len(cdImport.List) != 0 {
			for j := range cdImport.List {
				impDef := &cdImport.List[j]

				if impDef.ConfigMapRef != nil {
					if len(impDef.ConfigMapRef.Namespace) == 0 {
						impDef.ConfigMapRef.Namespace = obj.GetNamespace()
					}
				}
				if impDef.SecretRef != nil {
					if len(impDef.SecretRef.Namespace) == 0 {
						impDef.SecretRef.Namespace = obj.GetNamespace()
					}
				}
				if impDef.Ref != nil {
					if impDef.Ref.RepositoryContext == nil {
						if obj.Spec.ComponentDescriptor != nil && obj.Spec.ComponentDescriptor.Reference != nil {
							impDef.Ref.RepositoryContext = obj.Spec.ComponentDescriptor.Reference.RepositoryContext
						}
					}
				}
			}
		}
	}
}
