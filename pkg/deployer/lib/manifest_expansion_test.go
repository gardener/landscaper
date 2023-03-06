package lib

import (
	"encoding/json"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		manifests := []*runtime.RawExtension{
			raw(buildConfigMap("cm1")),
			raw(buildConfigMapList("cm2", "cm3")),
			raw(buildConfigMapList("cm4", "cm5", "cm6")),
			raw(buildConfigMap("cm7")),
			raw(buildConfigMapList("cm8")),
		}

		expanded, err := ExpandManifests(manifests)

		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(Equal([]*runtime.RawExtension{
			raw(buildConfigMap("cm1")),
			raw(buildConfigMap("cm2")),
			raw(buildConfigMap("cm3")),
			raw(buildConfigMap("cm4")),
			raw(buildConfigMap("cm5")),
			raw(buildConfigMap("cm6")),
			raw(buildConfigMap("cm7")),
			raw(buildConfigMap("cm8")),
		}))
	})

	It("should handle an empty list as list", func() {
		manifests := []*runtime.RawExtension{
			raw(v1.ConfigMapList{Items: []v1.ConfigMap{}}),
		}

		expanded, err := ExpandManifests(manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(HaveLen(0))
	})

	It("should handle a normal object not as list", func() {
		manifests := []*runtime.RawExtension{
			raw(buildConfigMap("cm1")),
		}

		expanded, err := ExpandManifests(manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(Equal([]*runtime.RawExtension{
			raw(buildConfigMap("cm1")),
		}))
	})

	It("should expand managed resource manifests", func() {
		manifests := []managedresource.Manifest{
			{Policy: managedresource.ManagePolicy, Manifest: raw(buildConfigMap("cm1"))},
			{Policy: managedresource.KeepPolicy, Manifest: raw(buildConfigMapList("cm2", "cm3"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMapList("cm4", "cm5", "cm6"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm7"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMapList("cm8"))},
		}

		expanded, err := ExpandManagedResourceManifests(manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(Equal([]managedresource.Manifest{
			{Policy: managedresource.ManagePolicy, Manifest: raw(buildConfigMap("cm1"))},
			{Policy: managedresource.KeepPolicy, Manifest: raw(buildConfigMap("cm2"))},
			{Policy: managedresource.KeepPolicy, Manifest: raw(buildConfigMap("cm3"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm4"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm5"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm6"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm7"))},
			{Policy: managedresource.ImmutablePolicy, Manifest: raw(buildConfigMap("cm8"))},
		}))
	})

	It("should not change managed resource manifests without lists", func() {
		manifests := []managedresource.Manifest{
			{Policy: managedresource.ManagePolicy, Manifest: raw(buildConfigMap("cm1"))},
			{Policy: managedresource.KeepPolicy, Manifest: raw(buildConfigMap("cm2"))},
		}

		expanded, err := ExpandManagedResourceManifests(manifests)
		Expect(err).NotTo(HaveOccurred())
		Expect(expanded).To(Equal(manifests))
	})
})
