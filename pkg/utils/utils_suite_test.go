// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	utils2 "github.com/gardener/landscaper/pkg/utils"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lscutils "github.com/gardener/landscaper/controller-utils/pkg/landscaper"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Test Suite")
}

var _ = Describe("Utils", func() {

	Context("References", func() {

		var (
			ctx   context.Context
			state *envtest.State
		)

		BeforeEach(func() {
			ctx = context.Background()
			var err error
			_, state, err = envtest.NewFakeClientFromPath("")
			utils.ExpectNoError(err)
			ns := &corev1.Namespace{}
			ns.GenerateName = "tests-"
			utils.ExpectNoError(state.Create(ctx, ns))
			state.Namespace = ns.Name
		})

		AfterEach(func() {
			defer ctx.Done()
		})

		It("should correctly resolve secret references with and without key", func() {
			type MyStruct struct {
				Foo lsv1alpha1.NamedObjectReference `json:"foo"`
			}
			key := "foo"

			var err error
			kSecret := &corev1.Secret{}
			kSecret.Name = "my-secret"
			kSecret.Namespace = state.Namespace
			kSecret.Data = map[string][]byte{}
			kSecret.Data[key], err = yaml.ToJSON([]byte(`name: foo
ref:
  name: bar
  namespace: baz`))
			utils.ExpectNoError(err)
			Expect(state.Create(ctx, kSecret)).To(Succeed())

			By("secret reference with key")
			sr1 := &lsv1alpha1.SecretReference{
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      kSecret.Name,
					Namespace: kSecret.Namespace,
				},
				Key: key,
			}

			wholeSecret1, value1, gen1, err := lscutils.ResolveSecretReference(ctx, state.Client, sr1)
			utils.ExpectNoError(err)

			ms1 := &MyStruct{
				Foo: lsv1alpha1.NamedObjectReference{},
			}
			utils.ExpectNoError(json.Unmarshal(value1, &ms1.Foo))

			By("secret reference without key")
			sr2 := &lsv1alpha1.SecretReference{
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      kSecret.Name,
					Namespace: kSecret.Namespace,
				},
			}

			wholeSecret2, value2, gen2, err := lscutils.ResolveSecretReference(ctx, state.Client, sr2)
			utils.ExpectNoError(err)

			ms2 := &MyStruct{}
			utils.ExpectNoError(json.Unmarshal(value2, ms2))

			Expect(wholeSecret2).To(BeEquivalentTo(wholeSecret1), "the whole returned secret content should be equivalent, independent of whether the reference contains a key or not")
			Expect(gen2).To(Equal(gen1), "the returned secret generation should be equivalent, independent of whether the reference contains a key or not")
			Expect(ms2).To(Equal(ms1), "unmarshalled objects should be identical, independent of whether the reference contains a key or not")
			Expect(ms2.Foo.Reference.Name).To(BeEquivalentTo("bar"))
		})

	})

})

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())

	ns := &corev1.Namespace{}
	ns.Name = utils2.GetCurrentPodNamespace()
	err = testenv.Client.Get(context.Background(), client.ObjectKeyFromObject(ns), ns)
	Expect(err == nil || apierrors.IsNotFound(err)).To(BeTrue())
	if err != nil && apierrors.IsNotFound(err) {
		Expect(testenv.Client.Create(context.Background(), ns)).ToNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})
