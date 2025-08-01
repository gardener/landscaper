package v1alpha1_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
)

var _ = Describe("Target", func() {

	It("should marshal a target with inline kubeconfig", func() {
		targetConfig := &targettypes.KubernetesClusterTargetConfig{
			Kubeconfig: targettypes.ValueRef{
				StrVal: ptr.To("a: 1\nb: 2"),
			},
		}
		targetConfigJSON, err := json.Marshal(targetConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(targetConfigJSON).To(MatchJSON(`{"kubeconfig":"a: 1\nb: 2"}`))

		target := &v1alpha1.Target{
			Spec: v1alpha1.TargetSpec{
				Type: targettypes.KubernetesClusterTargetType,
				Configuration: &v1alpha1.AnyJSON{
					RawMessage: targetConfigJSON,
				},
			},
		}
		targetJSON, err := json.Marshal(target)
		Expect(err).NotTo(HaveOccurred())

		Expect(targetJSON).To(MatchJSON(`{"metadata":{"creationTimestamp":null},"spec":{"type":"landscaper.gardener.cloud/kubernetes-cluster","config":{"kubeconfig":"a: 1\nb: 2"}}}`))
	})

	It("should unmarshal a target with inline kubeconfig", func() {
		targetJSON := []byte(`{"metadata":{"creationTimestamp":null},"spec":{"type":"landscaper.gardener.cloud/kubernetes-cluster","config":{"kubeconfig":"a: 1\nb: 2"}}}`)
		target := &v1alpha1.Target{}
		Expect(json.Unmarshal(targetJSON, target)).To(Succeed())

		configJSON := target.Spec.Configuration.RawMessage
		config := &targettypes.KubernetesClusterTargetConfig{}
		Expect(json.Unmarshal(configJSON, config)).To(Succeed())
		Expect(config).To(Equal(&targettypes.KubernetesClusterTargetConfig{
			Kubeconfig: targettypes.ValueRef{
				StrVal: ptr.To("a: 1\nb: 2"),
			},
		}))
	})
})
