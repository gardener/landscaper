// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	lsinstall "github.com/gardener/landscaper/apis/core/install"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Landscaper Controller Command Test Suite")
}

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Landscaper Controller", func() {

	Context("Deployer Bootstrap", func() {

		var (
			mgr   manager.Manager
			state *envtest.State
		)

		BeforeEach(func() {
			var (
				ctx = context.Background()
				err error
			)
			defer ctx.Done()
			mgr, err = manager.New(testenv.Env.Config, manager.Options{
				Metrics:   metricsserver.Options{BindAddress: "0"},
				NewClient: lsutils.NewUncachedClient(lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault),
			})
			Expect(err).ToNot(HaveOccurred())
			lsinstall.Install(mgr.GetScheme())

			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			ctx := context.Background()
			Expect(state.CleanupState(ctx)).To(Succeed())
		})
	})
})
