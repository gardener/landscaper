// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helmrepo

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Chart Repository Test Suite")
}

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).NotTo(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).NotTo(HaveOccurred())
})
