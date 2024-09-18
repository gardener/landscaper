package targettypes_test

import (
	"encoding/json"
	"testing"

	"k8s.io/utils/ptr"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Types Testing")
}

var _ = Describe("Kubernetes Cluster Target Types", func() {

	It("should marshal a kubeconfig", func() {
		targetConfig := &targettypes.KubernetesClusterTargetConfig{
			Kubeconfig: targettypes.ValueRef{
				StrVal: ptr.To("test-kubeconfig"),
			},
		}
		targetConfigJSON, err := json.Marshal(targetConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(targetConfigJSON).To(MatchJSON(`{"kubeconfig":"test-kubeconfig"}`))
	})

	It("should unmarshal a kubeconfig", func() {
		configJSON := []byte(`{"kubeconfig":"test-kubeconfig"}`)
		config := &targettypes.KubernetesClusterTargetConfig{}
		Expect(json.Unmarshal(configJSON, config)).To(Succeed())
		Expect(config).To(Equal(&targettypes.KubernetesClusterTargetConfig{
			Kubeconfig: targettypes.ValueRef{
				StrVal: ptr.To("test-kubeconfig"),
			},
		}))
	})

	It("should marshal an oidc config", func() {
		targetConfig := &targettypes.KubernetesClusterTargetConfig{
			OIDCConfig: &targettypes.OIDCConfig{
				Server: "test-server",
				CAData: []byte("test-cert"),
				ServiceAccount: v1alpha1.ObjectReference{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Audience: []string{"test-audience"},
			},
		}
		targetConfigJSON, err := json.Marshal(targetConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(targetConfigJSON).To(MatchJSON(`{"kubeconfig":null,"oidcConfig":{"server":"test-server","caData":"dGVzdC1jZXJ0","serviceAccount":{"name":"test-account","namespace":"test-namespace"},"audience":["test-audience"]}}`))
	})

	It("should unmarshal an oidc config", func() {
		configJSON := []byte(`{"kubeconfig":{},"oidcConfig":{"server":"test-server","caData":"dGVzdC1jZXJ0","serviceAccount":{"name":"test-account","namespace":"test-namespace"},"audience":["test-audience"]}}`)
		config := &targettypes.KubernetesClusterTargetConfig{}
		Expect(json.Unmarshal(configJSON, config)).To(Succeed())
		Expect(config).To(Equal(&targettypes.KubernetesClusterTargetConfig{
			Kubeconfig: targettypes.ValueRef{
				StrVal: ptr.To("{}"),
			},
			OIDCConfig: &targettypes.OIDCConfig{
				Server: "test-server",
				CAData: []byte("test-cert"),
				ServiceAccount: v1alpha1.ObjectReference{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Audience: []string{"test-audience"},
			},
		}))
	})
})
