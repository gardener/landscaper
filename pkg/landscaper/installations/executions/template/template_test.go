// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
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
	var stateHandler GenericStateHandler

	BeforeEach(func() {
		stateHandler = NewMemoryStateHandler()
	})

	Context("TemplateDeployExecutions", func() {
		It("should return the raw template if no templating funcs are defined", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-01.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{}, stateHandler)

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
			op := New(&installations.Operation{}, stateHandler)

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
			op := New(&installations.Operation{}, stateHandler)

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
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
			op := New(&installations.Operation{}, stateHandler)
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
			op := New(&installations.Operation{}, stateHandler)

			_, err = op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, map[string]interface{}{"version": "0.0.0"})
			Expect(err).To(HaveOccurred())
		})

		It("should use the state to template", func() {
			tmpl, err := ioutil.ReadFile(filepath.Join(testdataDir, "template-09.yaml"))
			Expect(err).ToNot(HaveOccurred())
			exec := make([]lsv1alpha1.TemplateExecutor, 0)
			Expect(yaml.Unmarshal(tmpl, &exec)).ToNot(HaveOccurred())

			blue := &lsv1alpha1.Blueprint{}
			blue.DeployExecutions = exec
			op := New(&installations.Operation{}, stateHandler)

			_, err = op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, map[string]interface{}{"version": "0.0.1"})
			Expect(err).ToNot(HaveOccurred())

			state := map[string]string{
				"version": "0.0.2",
			}
			stateBytes, err := json.Marshal(state)
			Expect(err).ToNot(HaveOccurred())
			Expect(stateHandler.Store(context.TODO(), exec[0].Name, stateBytes)).To(Succeed())
			_, err = op.TemplateDeployExecutions(&blueprints.Blueprint{
				Info: blue,
				Fs:   nil,
			}, &cdutils.ResolvedComponentDescriptor{}, map[string]interface{}{"version": "0.0.2"})

			Expect(err).ToNot(HaveOccurred())
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
			op := New(&installations.Operation{}, stateHandler)

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
			op := New(&installations.Operation{}, stateHandler)

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
			op := New(&installations.Operation{}, stateHandler)

			memFs := memoryfs.New()
			err = vfs.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
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
