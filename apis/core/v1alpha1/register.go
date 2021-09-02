// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha1 is the v1alpha1 version of the API.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	lsschema "github.com/gardener/landscaper/apis/schema"

	"github.com/gardener/landscaper/apis/core"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: core.GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder is a new Schema Builder which registers our API.
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs, addConversionFuncs)
	localSchemeBuilder = &SchemeBuilder
	// AddToScheme is a reference to the Schema Builder's AddToScheme function.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Schema.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DataObject{},
		&DataObjectList{},
		&Target{},
		&TargetList{},
		&Blueprint{},
		&InstallationTemplate{},
		&Installation{},
		&InstallationList{},
		&Execution{},
		&ExecutionList{},
		&DeployItem{},
		&DeployItemList{},
		&Context{},
		&ContextList{},
		&ComponentOverwrites{},
		&ComponentOverwritesList{},
		&Environment{},
		&EnvironmentList{},
		&DeployerRegistration{},
		&DeployerRegistrationList{},
	)
	if err := RegisterConversions(scheme); err != nil {
		return err
	}
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// ResourceDefinition defines the custom resources of this version.
var ResourceDefinition = func() lsschema.CustomResourceDefinitions {
	return lsschema.CustomResourceDefinitions{
		Group:     SchemeGroupVersion.Group,
		Version:   SchemeGroupVersion.Version,
		OutputDir: "../pkg/landscaper/crdmanager/crdresources",

		Definitions: []lsschema.CustomResourceDefinition{
			InstallationDefinition,
			ExecutionDefinition,
			DeployItemDefinition,
			DataObjectDefinition,
			TargetDefinition,
			ContextDefinition,
			DeployerRegistrationDefinition,
			EnvironmentDefinition,
			ComponentOverwritesDefinition,
		},
	}
}()
