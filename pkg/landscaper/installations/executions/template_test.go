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

package executions

import (
	"io/ioutil"
	"os"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	vfsutil "github.com/gardener/landscaper/pkg/utils/ioutil"
)

var _ = Describe("Template", func() {

	It("should return the raw template if no templating funcs are defined", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-01.yaml")
		Expect(err).ToNot(HaveOccurred())
		expect := make([]lsv1alpha1.ExecutionItem, 0)
		Expect(yaml.Unmarshal(tmpl, &expect)).ToNot(HaveOccurred())

		blue := &lsv1alpha1.Blueprint{}
		blue.Executors = string(tmpl)
		op := New(&installations.Operation{})

		res, err := op.template(&blueprints.Blueprint{
			Info: blue,
			Fs:   nil,
		}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(ConsistOf(expect))
	})

	It("should use the import values to template", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-02.yaml")
		Expect(err).ToNot(HaveOccurred())

		blue := &lsv1alpha1.Blueprint{}
		blue.Executors = string(tmpl)
		op := New(&installations.Operation{})

		res, err := op.template(&blueprints.Blueprint{
			Info: blue,
			Fs:   nil,
		}, map[string]string{"version": "0.0.0"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))

		config := make(map[string]interface{})
		Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
		Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
	})

	It("should read the content of a file to template", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-03.yaml")
		Expect(err).ToNot(HaveOccurred())

		blue := &lsv1alpha1.Blueprint{}
		blue.Executors = string(tmpl)
		op := New(&installations.Operation{})

		memFs := memoryfs.New()
		err = vfsutil.WriteFile(memFs, "VERSION", []byte("0.0.0"), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())

		res, err := op.template(&blueprints.Blueprint{
			Info: blue,
			Fs:   memFs,
		}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))

		config := make(map[string]interface{})
		Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
		Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
	})

	It("should use a resource from the component descriptor", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-04.yaml")
		Expect(err).ToNot(HaveOccurred())

		blue := &lsv1alpha1.Blueprint{}
		blue.Executors = string(tmpl)
		op := New(&installations.Operation{})
		op.ResolvedComponentDescriptor = cdv2.ComponentDescriptorList{
			Components: []cdv2.ComponentDescriptor{
				{
					ComponentSpec: cdv2.ComponentSpec{
						ObjectMeta: cdv2.ObjectMeta{
							Name:    "mycomp",
							Version: "1.0.0",
						},
						ExternalResources: []cdv2.Resource{
							{
								ObjectMeta: cdv2.ObjectMeta{
									Name:    "mycustomimage",
									Version: "1.0.0",
								},
								Access: &cdv2.OCIRegistryAccess{
									ObjectType: cdv2.ObjectType{
										Type: cdv2.OCIRegistryType,
									},
									ImageReference: "quay.io/example/myimage:1.0.0",
								},
							},
						},
					},
				},
			},
		}

		res, err := op.template(&blueprints.Blueprint{
			Info: blue,
		}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))

		config := make(map[string]interface{})
		Expect(yaml.Unmarshal(res[0].Configuration.Raw, &config)).ToNot(HaveOccurred())
		Expect(config).To(HaveKeyWithValue("image", "quay.io/example/myimage:1.0.0"))
	})

})
