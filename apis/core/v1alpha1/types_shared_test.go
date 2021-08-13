// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"sigs.k8s.io/yaml"
)

var _ = Describe("Shared Types", func() {

	Context("ExportDefinitions", func() {
		It("parsing null should result in empty raw message", func() {

			executor1 := TemplateExecutor{}
			result1, err := yaml.Marshal(executor1)

			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(result1).NotTo(gomega.HaveLen(0))

			executor2 := &TemplateExecutor{}
			err = yaml.Unmarshal(result1, executor2)

			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(executor2.Template.RawMessage).To(gomega.BeNil())
		})

	})

})
