// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targettypes

import (
	"encoding/json"

	"k8s.io/utils/ptr"

	"github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// KubernetesClusterTargetType defines the landscaper kubernetes cluster target.
const KubernetesClusterTargetType v1alpha1.TargetType = core.GroupName + "/kubernetes-cluster"

// KubernetesClusterTargetConfig defines the landscaper kubernetes cluster target config.
type KubernetesClusterTargetConfig struct {
	// Kubeconfig defines kubeconfig as string.
	Kubeconfig ValueRef `json:"kubeconfig"`
}

// DefaultKubeconfigKey is the default that is used to hold a kubeconfig.
const DefaultKubeconfigKey = "kubeconfig"

// ValueRef holds a value that can be either defined by string or by a secret ref.
type ValueRef struct {
	StrVal *string `json:"-"`
}

// kubeconfigJSON is a helper struct for decoding.
type kubeconfigJSON struct {
	Kubeconfig *ValueRef `json:"kubeconfig"`
}

// MarshalJSON implements the json marshaling for a JSON
func (v ValueRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.StrVal)
}

// UnmarshalJSON implements json unmarshaling for a JSON
func (v *ValueRef) UnmarshalJSON(data []byte) error {
	// parse into string instead
	var strVal string
	err := json.Unmarshal(data, &strVal)
	if err == nil {
		v.StrVal = &strVal
		return nil
	}
	v.StrVal = ptr.To(string(data))
	return nil
}

func (kc *KubernetesClusterTargetConfig) UnmarshalJSON(data []byte) error {
	kj := &kubeconfigJSON{}
	err := json.Unmarshal(data, kj)
	if err == nil && kj.Kubeconfig != nil {
		// parsing was successful
		kc.Kubeconfig = *kj.Kubeconfig
		return nil
	}
	return kc.Kubeconfig.UnmarshalJSON(data)
}

func (v ValueRef) OpenAPISchemaType() []string {
	return []string{
		"object",
		"string",
	}
}
func (v ValueRef) OpenAPISchemaFormat() string { return "" }
