// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/integration/tutorial"
)

var opts *framework.Options

func init() {
	opts = &framework.Options{}
	opts.AddFlags(flag.CommandLine)
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)

	opts.RootPath = "../../"
	f, err := framework.New(opts)
	Expect(err).ToNot(HaveOccurred())

	// todo: register tests
	tutorial.RegisterTests(f)

	AfterSuite(func() {
		f.Cleanup.Run()
	})

	RunSpecs(t, "Landscaper Integration Test Suite")
}
