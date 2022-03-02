// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

var _ = Describe("TemplateInputFormatter", func() {

	It("should format input parameters", func() {
		i := map[string]interface{}{
			"myobj": map[string]interface{}{
				"myvar": "inner",
			},
			"mystring": "val",
			"myint":    42,
		}

		f := template.NewTemplateInputFormatter(i, false)
		formatted := f.Format("\t")
		Expect(formatted).To(ContainSubstring("\tmyobj: {\"myvar\":\"inner\"}\n"))
		Expect(formatted).To(ContainSubstring("\tmystring: \"val\"\n"))
		Expect(formatted).To(ContainSubstring("\tmyint: 42\n"))
	})

	It("should hide sensitive data in imports", func() {
		i := map[string]interface{}{
			"myobj": map[string]interface{}{
				"myvar": "inner",
			},
			"mystring": "val",
			"myint":    42,
		}

		f := template.NewTemplateInputFormatter(i, false, "myobj", "myint")
		formatted := f.Format("\t")
		Expect(formatted).To(ContainSubstring("\tmyobj: {\"myvar\":\"[...] (string)\"}\n"))
		Expect(formatted).To(ContainSubstring("\tmystring: \"val\"\n"))
		Expect(formatted).To(ContainSubstring("\tmyint: \"[...] (int)\"\n"))

		myobj, ok := i["myobj"].(map[string]interface{})
		Expect(ok).To(BeTrue())
		myvar, ok := myobj["myvar"].(string)
		Expect(ok).To(BeTrue())
		Expect(myvar).To(Equal("inner"))

		myint, ok := i["myint"].(int)
		Expect(ok).To(BeTrue())
		Expect(myint).To(Equal(42))
	})

	It("should compress large keys", func() {
		large := strings.Repeat("a", 1024)
		i := map[string]interface{}{
			"large": large,
		}
		f := template.NewTemplateInputFormatter(i, false)
		formatted := f.Format("")

		r, err := regexp.Compile(`large: >gzip>base64> (\S+)`)
		Expect(err).NotTo(HaveOccurred())

		m := r.FindStringSubmatch(formatted)
		Expect(m).NotTo(BeNil())
		Expect(m).To(HaveLen(2))

		compressed, err := base64.StdEncoding.DecodeString(m[1])
		Expect(err).NotTo(HaveOccurred())

		gz, err := gzip.NewReader(bytes.NewReader(compressed))
		Expect(err).ToNot(HaveOccurred())

		decompressed, err := io.ReadAll(gz)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(decompressed)).To(Equal(fmt.Sprintf("\"%s\"", large)))

		gz.Close()
	})

	It("should pretty print input parameters", func() {
		i := map[string]interface{}{
			"myobj": map[string]interface{}{
				"myvar": "inner",
			},
			"mystring": "val",
			"myint":    42,
		}

		f := template.NewTemplateInputFormatter(i, true)
		formatted := f.Format("\t")
		Expect(formatted).To(ContainSubstring("\tmyobj: {\n\t  \"myvar\": \"inner\"\n\t}\n"))
	})
})
