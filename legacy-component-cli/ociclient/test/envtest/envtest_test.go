// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest_test

import (
	"context"
	"path/filepath"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/test/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "envtest Test Suite")
}

var _ = ginkgo.Describe("Test Environment", func() {

	ginkgo.It("should run and stop a test registry", func() {
		ctx := context.Background()
		defer ctx.Done()
		testenv := envtest.New(envtest.Options{
			RegistryBinaryPath: filepath.Join("../../../", envtest.DefaultRegistryBinaryPath),
		})
		Expect(testenv.Start(ctx)).To(Succeed())

		Expect(testenv.Close()).To(Succeed())
	})

})
