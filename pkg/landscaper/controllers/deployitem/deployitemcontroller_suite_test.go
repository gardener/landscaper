// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitem_test

import (
	"path/filepath"
	"testing"
	"time"

	lscore "github.com/gardener/landscaper/apis/core"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Item Controller Test Suite")
}

var (
	testPickupTimeoutDuration = lscore.Duration{
		Duration: 10 * time.Second,
	}
	testProgressingTimeoutDuration = lscore.Duration{
		Duration: 30 * time.Second,
	}
)

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../../")
	testdataDir = filepath.Join("./testdata/")
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
