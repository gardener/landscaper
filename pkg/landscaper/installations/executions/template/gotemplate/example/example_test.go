// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate_test

import (
	"fmt"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
)

var _ = Describe("Templating Examples", func() {

	testdataDir := filepath.Join(".", "testdata")

	It("should render a simple go template", func() {

		tmpl, err := os.ReadFile(filepath.Join(testdataDir, "test1", "template.yaml"))
		Expect(err).NotTo(HaveOccurred())

		bp := blueprints.New(nil, memoryfs.New())
		t := gotemplate.NewTemplateExecution(bp, nil, nil, nil, nil)
		values := map[string]interface{}{
			"values": map[string]interface{}{
				"items": []string{"A", "B", "C"},
			},
		}
		result, err := t.Execute(string(tmpl), values)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).ToNot(Equal("A-B-C"))
	})

	It("should render a go template (test2)", func() {

		tmpl, err := os.ReadFile(filepath.Join(testdataDir, "test2", "template.yaml"))
		Expect(err).NotTo(HaveOccurred())
		cdBytes, err := os.ReadFile(filepath.Join(testdataDir, "test2", "component-descriptor.yaml"))
		Expect(err).NotTo(HaveOccurred())

		bp := blueprints.New(nil, memoryfs.New())
		t := gotemplate.NewTemplateExecution(bp, nil, nil, nil, nil)

		cd := map[string]interface{}{}
		Expect(yaml.Unmarshal(cdBytes, &cd)).To(Succeed())
		values := cd

		result, err := t.Execute(string(tmpl), values)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeEmpty())
		fmt.Println(string(result))
	})

	It("should render a go template (test3)", func() {

		tmpl, err := os.ReadFile(filepath.Join(testdataDir, "test3", "template.yaml"))
		Expect(err).NotTo(HaveOccurred())

		cdMonolithBytes, err := os.ReadFile(filepath.Join(testdataDir, "test3", "bizx-monolith.yaml"))
		Expect(err).NotTo(HaveOccurred())
		cdMonolith := cdv2.ComponentDescriptor{}
		Expect(yaml.Unmarshal(cdMonolithBytes, &cdMonolith)).To(Succeed())

		cdCoreBytes, err := os.ReadFile(filepath.Join(testdataDir, "test3", "bizx-core.yaml"))
		Expect(err).NotTo(HaveOccurred())
		cdCore := cdv2.ComponentDescriptor{}
		Expect(yaml.Unmarshal(cdCoreBytes, &cdCore)).To(Succeed())

		cdJanitorBytes, err := os.ReadFile(filepath.Join(testdataDir, "test3", "bizx-sidecar-janitor.yaml"))
		Expect(err).NotTo(HaveOccurred())
		cdJanitor := cdv2.ComponentDescriptor{}
		Expect(yaml.Unmarshal(cdJanitorBytes, &cdJanitor)).To(Succeed())

		cdKafkaBytes, err := os.ReadFile(filepath.Join(testdataDir, "test3", "kafka.yaml"))
		Expect(err).NotTo(HaveOccurred())
		cdKafka := cdv2.ComponentDescriptor{}
		Expect(yaml.Unmarshal(cdKafkaBytes, &cdKafka)).To(Succeed())

		cdList := &cdv2.ComponentDescriptorList{
			Components: []cdv2.ComponentDescriptor{cdMonolith, cdCore, cdJanitor, cdKafka},
		}

		bp := blueprints.New(nil, memoryfs.New())
		t := gotemplate.NewTemplateExecution(bp, &cdCore, cdList, nil, nil)

		blueprintExecutionOptionstemplate := template.NewBlueprintExecutionOptions(nil, nil, &cdMonolith, cdList, nil)
		values, err := (&blueprintExecutionOptionstemplate).Values()
		Expect(err).NotTo(HaveOccurred())

		result, err := t.Execute(string(tmpl), values)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeEmpty())
		fmt.Println(string(result))
	})

})
