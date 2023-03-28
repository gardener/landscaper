// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate_test

import (
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
)

var _ = Describe("TemplateDeployExecutions", func() {

	It("should render a simple go template", func() {
		fs := memoryfs.New()
		bp := blueprints.New(nil, fs)
		tmpl := "{{ .values.test }}"
		t := gotemplate.NewTemplateExecution(bp, nil, nil, nil)
		values := map[string]interface{}{
			"values": map[string]interface{}{
				"test": "foo",
			},
		}
		res, err := t.Execute(tmpl, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal([]byte("foo")))
	})

	It("should render a go template with an include function", func() {
		fs := memoryfs.New()
		Expect(vfs.WriteFile(fs, "template.include", []byte("{{ .values.test }}"), 0600)).To(Succeed())
		bp := blueprints.New(nil, fs)
		tmpl := `{{ include "template.include" . }}`
		t := gotemplate.NewTemplateExecution(bp, nil, nil, nil)
		values := map[string]interface{}{
			"values": map[string]interface{}{
				"test": "foo",
			},
		}
		res, err := t.Execute(tmpl, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal([]byte("foo")))
	})

	It("should render a go template with an include function in a pipe", func() {
		fs := memoryfs.New()
		Expect(vfs.WriteFile(fs, "template.include", []byte("value: {{ .values.test }}\nconst: bar"), 0600)).To(Succeed())
		bp := blueprints.New(nil, fs)
		tmpl := `config:
{{ include "template.include" . | indent 2 }}`
		t := gotemplate.NewTemplateExecution(bp, nil, nil, nil)
		values := map[string]interface{}{
			"values": map[string]interface{}{
				"test": "foo",
			},
		}
		res, err := t.Execute(tmpl, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(BeEquivalentTo("config:\n  value: foo\n  const: bar"))
	})

})
