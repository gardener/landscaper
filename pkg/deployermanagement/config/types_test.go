// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/deployermanagement/config"
)

var _ = Describe("types", func() {

	Context("Deployer Configuration", func() {

		It("should decode a deployer configuration with values", func() {
			data := []byte(`
Deployers:
  helm:
    somekey: someval
`)
			data, err := yaml.YAMLToJSON(data)
			Expect(err).ToNot(HaveOccurred())
			deployerConfig := &config.DeployersConfiguration{}
			Expect(json.Unmarshal(data, deployerConfig)).To(Succeed())

			Expect(deployerConfig.Deployers).To(HaveKey("helm"))
			Expect(deployerConfig.Deployers["helm"].Type).To(Equal(config.ValuesType))
			Expect(deployerConfig.Deployers["helm"].Values).To(HaveKeyWithValue("somekey", "someval"))
		})

	})

})
