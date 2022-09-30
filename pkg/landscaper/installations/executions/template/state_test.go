// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var (
	testenv *envtest.Environment
)

var _ = BeforeSuite(func() {
	var err error
	projectRoot := filepath.Join("../../../../../")
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("State", func() {

	Context("kubernetes handler", func() {

		It("should store state in a secret and read the same data from it", func() {
			ctx := context.Background()
			defer ctx.Done()
			stateHdlr := KubernetesStateHandler{
				KubeClient: testenv.Client,
				Inst: &lsv1alpha1.Installation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						UID:       types.UID("abc-abc-abc"),
					},
				},
			}

			data := []byte("my data")
			Expect(stateHdlr.Store(ctx, "my-exec", data)).To(Succeed())

			res, err := stateHdlr.Get(ctx, "my-exec")
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(data))
		})

	})

})
