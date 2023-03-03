package lib

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest expansion", func() {

	buildConfigMap := func(name string) v1.ConfigMap {
		return v1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "testNamespace"},
			Data:       map[string]string{"testKey": "testValue"},
		}
	}

	buildConfigMapList := func(names ...string) v1.ConfigMapList {
		items := make([]v1.ConfigMap, len(names))
		for i := range names {
			items[i] = buildConfigMap(names[i])
		}
		return v1.ConfigMapList{Items: items}
	}

	raw := func(a any) *runtime.RawExtension {
		bytes, err := json.Marshal(a)
		Expect(err).NotTo(HaveOccurred())
		return &runtime.RawExtension{Raw: bytes}
	}

	It("should expand manifests", func() {
		names := []string{"cm1", "cm2", "cm3", "cm4", "cm5", "cm6", "cm7", "cm8"}

		manifests := []*runtime.RawExtension{
			raw(buildConfigMap("cm1")),
			raw(buildConfigMapList("cm2", "cm3")),
			raw(buildConfigMapList("cm4", "cm5", "cm6")),
			raw(buildConfigMap("cm7")),
			raw(buildConfigMapList("cm8")),
		}

		expanded, err := ExpandManifests(manifests)

		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(HaveLen(8))

		for i, name := range names {
			cm := &v1.ConfigMap{}
			Expect(expanded[i]).NotTo(BeNil())
			Expect(json.Unmarshal(expanded[i].Raw, cm)).To(Succeed())
			Expect(cm.Name).To(Equal(name))
		}
	})
})
