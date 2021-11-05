// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsschema "github.com/gardener/landscaper/apis/schema"

	"github.com/gardener/landscaper/apis/core"
)

// TargetType defines the type of the target.
type TargetType string

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetList contains a list of Targets
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Target `json:"items"`
}

// TargetDefinition defines the Target resource CRD.
var TargetDefinition = lsschema.CustomResourceDefinition{
	Names: lsschema.CustomResourceDefinitionNames{
		Plural:   "targets",
		Singular: "target",
		ShortNames: []string{
			"tgt",
			"tg",
		},
		Kind: "Target",
	},
	Scope:             lsschema.NamespaceScoped,
	Storage:           true,
	Served:            true,
	SubresourceStatus: false,
	AdditionalPrinterColumns: []lsschema.CustomResourceColumnDefinition{
		{
			Name:     "Type",
			Type:     "string",
			JSONPath: ".spec.type",
		},
		{
			Name:     "Context",
			Type:     "string",
			JSONPath: ".metadata.labels['data\\.landscaper\\.gardener\\.cloud\\/context']",
		},
		{
			Name:     "Key",
			Type:     "string",
			JSONPath: ".metadata.labels['data\\.landscaper\\.gardener\\.cloud\\/key']",
		},
		{
			Name:     "Age",
			Type:     "date",
			JSONPath: ".metadata.creationTimestamp",
		},
	},
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Target defines a specific data object that defines target environment.
// Every deploy item can have a target which is used by the deployer to install the specific application.
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TargetSpec `json:"spec"`
}

// TargetSpec contains the definition of a target.
type TargetSpec struct {
	// Type is the type of the target that defines its data structure.
	// The actual schema may be defined by a target type crd in the future.
	Type TargetType `json:"type"`
	// Configuration contains the target type specific configuration.
	// +optional
	Configuration AnyJSON `json:"config,omitempty"`
}

// TargetTemplate exposes specific parts of a target that are used in the exports
// to export a target
type TargetTemplate struct {
	TargetSpec `json:",inline"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

//////////////////////////////
//     Target Types         //
//////////////////////////////
// todo: refactor to own package

// KubernetesClusterTargetType defines the landscaper kubernetes cluster target.
const KubernetesClusterTargetType TargetType = core.GroupName + "/kubernetes-cluster"

// KubernetesClusterTargetConfig defines the landscaper kubernetes cluster target config.
type KubernetesClusterTargetConfig struct {
	// Kubeconfig defines kubeconfig as string.
	Kubeconfig ValueRef `json:"kubeconfig"`
}

// DefaultKubeconfigKey is the default that is used to hold a kubeconfig.
const DefaultKubeconfigKey = "kubeconfig"

// ValueRef holds a value that can be either defined by string or by a secret ref.
type ValueRef struct {
	StrVal    *string          `json:"-"`
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// valueRefJSON is a helper struct to decode json into a secret ref object.
type valueRefJSON struct {
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// MarshalJSON implements the json marshaling for a JSON
func (v ValueRef) MarshalJSON() ([]byte, error) {
	if v.StrVal != nil {
		return json.Marshal(v.StrVal)
	}
	ref := valueRefJSON{
		SecretRef: v.SecretRef,
	}
	return json.Marshal(ref)
}

// UnmarshalJSON implements json unmarshaling for a JSON
func (v *ValueRef) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		var strVal string
		if err := json.Unmarshal(data, &strVal); err != nil {
			return err
		}
		v.StrVal = &strVal
		return nil
	}
	ref := &valueRefJSON{}
	if err := json.Unmarshal(data, ref); err != nil {
		return err
	}
	v.SecretRef = ref.SecretRef
	return nil
}

func (v ValueRef) OpenAPISchemaType() []string {
	return []string{
		"object",
		"string",
	}
}
func (v ValueRef) OpenAPISchemaFormat() string { return "" }
