// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	coctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/componentoverwrites"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component OVerwrites Controller Test Suite")
}

var (
	testenv *envtest.Environment
)

var _ = BeforeSuite(func() {
	var err error
	projectRoot := filepath.Join("../../../../")
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Reconcile", func() {

	var (
		ctx   context.Context
		state *envtest.State
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		defer ctx.Done()
		if state != nil {
			Expect(state.CleanupState(ctx))
		}
	})

	It("should add a component overwrite to the manager", func() {
		mgr := componentoverwrites.New()
		c := coctrl.NewController(logging.Discard(), testenv.Client, api.LandscaperScheme, mgr)
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())

		co := &lsv1alpha1.ComponentOverwrites{}
		co.Name = "my-co"
		co.Overwrites = lsv1alpha1.ComponentOverwriteList{
			{
				Component: lsv1alpha1.ComponentOverwriteReference{
					ComponentName: "my-comp",
				},
				Target: lsv1alpha1.ComponentOverwriteReference{
					ComponentName: "0.0.0",
				},
			},
		}
		Expect(state.Create(ctx, co)).To(Succeed())

		testutil.ShouldReconcile(ctx, c, testutil.Request("my-co", ""))

		ref := &lsv1alpha1.ComponentDescriptorReference{}
		ref.ComponentName = "my-comp"
		ov, err := mgr.Replace(ref)
		Expect(err).ToNot(HaveOccurred())
		Expect(ov).To(BeTrue())
	})

})
