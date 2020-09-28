// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	vfsutil "github.com/gardener/landscaper/pkg/utils/ioutil"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Executions Test Suite")
}

var _ = Describe("TemplateDeployExecutions", func() {

	Context("GoTemplate", func() {
		testdataDir := filepath.Join("./testdata", "gotemplate")
		runTestSuite(testdataDir)
	})

	Context("Spiff", func() {
		testdataDir := filepath.Join("./testdata", "spifftemplate")
		runTestSuite(testdataDir)
	})

})

func runTestSuite(testdataDir string) {
	Context("TemplateDeployExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-01.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{})

			res, err := op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("init"),
				"Type": Equal(lsv1alpha1.ExecutionType("container")),
			}))
		})

		It("should use the import values to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-02.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{})

			res, err := op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, map[string]interface{}{"version": "0.0.0"})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-03.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{})

			memFs := memoryfs.New()
			err = vfsutil.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   memFs,
			}, &cdutils.ResolvedComponentDescriptor{}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should use a resource from the component descriptor", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-04.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{})
			cd := &cdutils.ResolvedComponentDescriptor{
				ResolvedComponentSpec: cdutils.ResolvedComponentSpec{
					ObjectMeta: cdv2.ObjectMeta{
						Name:    "mycomp",
						Version: "1.0.0",
					},
					ExternalResources: map[string]cdv2.Resource{
						"mycustomimage": {
							ObjectMeta: cdv2.ObjectMeta{
								Name:    "mycustomimage",
								Version: "1.0.0",
							},
							TypedObjectAccessor: cdv2.NewTypeOnly(cdv2.OCIImageType),
							Access: &cdv2.OCIRegistryAccess{
								ObjectType: cdv2.ObjectType{
									Type: cdv2.OCIRegistryType,
								},
								ImageReference: "quay.io/example/myimage:1.0.0",
							},
						},
					},
				},
			}

			res, err := op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
			}, cd, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))

			config := make(map[string]interface{})
			Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
		})

		It("should throw an error when the template tries to template a undefined value", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-08.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{})

			_, err = op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, map[string]interface{}{"version": "0.0.0"})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("TemplateExportExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-05.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(&installations.Operation{})

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("testKey", "myval"))
		})

		It("should use the export values to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-06.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(&installations.Operation{})

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, map[string]interface{}{"version": "0.0.0"})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})

		It("should read the content of a file to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-07.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.ExportExecutions = exec
			op := New(&installations.Operation{})

			memFs := memoryfs.New()
			err = vfsutil.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			res, err := op.TemplateExportExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   memFs,
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(1))
			Expect(res).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
		})
	})
}
