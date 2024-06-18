// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	testutils2 "github.com/gardener/landscaper/pkg/components/testutils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/test/utils/envtest"

	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Executions Test Suite")
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

func createDefaultContextsForNamespace(kubeClient client.Client) {
	// create default repo for all namespaces
	repoCtx := testutils2.NewLocalRepository("../testdata/registry")
	list, err := os.ReadDir("./testdata/state")
	Expect(err).To(Succeed())
	for _, d := range list {
		Expect(testutils.CreateDefaultContext(context.TODO(), kubeClient, repoCtx, filepath.Base(d.Name()))).To(Succeed())
	}
}
