// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/template"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Template Test Suite")
}

var _ = ginkgo.Describe("Template", func() {

	ginkgo.Context("Parse Arguments", func() {

		ginkgo.It("should parse one argument after a '--'", func() {
			opts := template.Options{}
			Expect(opts.Parse([]string{"MY_VAR=test"})).To(BeNil())
			Expect(opts.Vars).To(HaveKeyWithValue("MY_VAR", "test"))
		})

		ginkgo.It("should return non variable arguments", func() {
			opts := template.Options{}

			args := opts.Parse([]string{"--", "MY_VAR=test", "my-arg"})
			Expect(args).To(Equal([]string{
				"--", "my-arg",
			}))
			Expect(opts.Vars).To(HaveKeyWithValue("MY_VAR", "test"))
		})

		ginkgo.It("should parse multiple values", func() {
			opts := template.Options{}
			Expect(opts.Parse([]string{"MY_VAR=test", "myOtherVar=true"})).To(BeNil())
			Expect(opts.Vars).To(HaveKeyWithValue("MY_VAR", "test"))
			Expect(opts.Vars).To(HaveKeyWithValue("myOtherVar", "true"))
		})

	})

	ginkgo.Context("Template", func() {
		ginkgo.It("should template with a single value", func() {
			s := "my ${MY_VAR}"
			opts := template.Options{}
			opts.Vars = map[string]string{
				"MY_VAR": "test",
			}
			res, err := opts.Template(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal("my test"))
		})

		ginkgo.It("should template multiple value", func() {
			s := "my ${MY_VAR} ${my_second_var}"
			opts := template.Options{}
			opts.Vars = map[string]string{
				"MY_VAR":        "test",
				"my_second_var": "testvalue",
			}
			res, err := opts.Template(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal("my test testvalue"))
		})

		ginkgo.It("should use an empty string if no value is provided", func() {
			s := "my ${MY_VAR}"
			opts := template.Options{}
			opts.Vars = map[string]string{}
			res, err := opts.Template(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal("my "))
		})

	})

})
