// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targettypes

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
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

	OIDCConfig *OIDCConfig `json:"oidcConfig,omitempty"`
}

// DefaultKubeconfigKey is the default that is used to hold a kubeconfig.
const DefaultKubeconfigKey = "kubeconfig"

// ValueRef holds a value that can be either defined by string or by a secret ref.
type ValueRef struct {
	StrVal *string `json:"-"`
}

// kubeconfigJSON is a helper struct for decoding.
type kubeconfigJSON struct {
	Kubeconfig *ValueRef   `json:"kubeconfig"`
	OIDCConfig *OIDCConfig `json:"oidcConfig,omitempty"`
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
	if err == nil && (kj.Kubeconfig != nil || kj.OIDCConfig != nil) {
		// parsing was successful
		if kj.Kubeconfig != nil {
			kc.Kubeconfig = *kj.Kubeconfig
		}
		kc.OIDCConfig = kj.OIDCConfig
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

type OIDCConfig struct {
	Server            string                  `json:"server,omitempty"`
	CAData            []byte                  `json:"caData,omitempty"`
	ServiceAccount    v1.LocalObjectReference `json:"serviceAccount,omitempty"`
	Audience          []string                `json:"audience,omitempty"`
	ExpirationSeconds *int64                  `json:"expirationSeconds,omitempty"`
}
