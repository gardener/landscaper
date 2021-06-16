// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/deployermanagement/config"
)

var _ = Describe("Landscaper Controller", func() {

	Context("Options", func() {

		It("should parse enabled deployers", func() {
			opts := &config.Options{}
			opts.Deployers = "deployer1,deployer2"
			Expect(opts.Complete()).To(Succeed())

			Expect(opts.EnabledDeployers).To(ConsistOf("deployer1", "deployer2"))
		})

	})
})
