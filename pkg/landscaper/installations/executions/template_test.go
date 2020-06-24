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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

var _ = Describe("Template", func() {

	It("should return the raw template if no templating funcs are defined", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-01.yaml")
		Expect(err).ToNot(HaveOccurred())
		expect := make([]lsv1alpha1.ExecutionItem, 0)
		Expect(yaml.Unmarshal(tmpl, &expect)).ToNot(HaveOccurred())

		rawDef := &lsv1alpha1.ComponentDefinition{}
		rawDef.Executors = string(tmpl)
		op := New(nil)

		res, err := op.template(rawDef, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(ConsistOf(expect))
	})

	It("should return the raw template if no templating funcs are defined", func() {
		tmpl, err := ioutil.ReadFile("./testdata/template-02.yaml")
		Expect(err).ToNot(HaveOccurred())
		expect := make([]lsv1alpha1.ExecutionItem, 0)
		Expect(yaml.Unmarshal(tmpl, &expect)).ToNot(HaveOccurred())

		rawDef := &lsv1alpha1.ComponentDefinition{}
		rawDef.Executors = string(tmpl)
		op := New(nil)

		res, err := op.template(rawDef, map[string]string{"version": "0.0.0"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(1))

		config := make(map[string]interface{})
		Expect(yaml.Unmarshal(res[0].Configuration, &config)).ToNot(HaveOccurred())
		Expect(config).To(HaveKeyWithValue("image", "my-custom-image:0.0.0"))
	})

})
