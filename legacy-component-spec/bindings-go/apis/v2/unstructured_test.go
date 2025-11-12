// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package v2_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

var _ = Describe("UnstructuredTypedObject", func() {

	It("should create a new unstructured object using a typed accessor", func() {
		type example struct {
			v2.ObjectType
			Test int `json:"test"`
		}
		obj := example{}
		obj.Type = "example"
		obj.Test = 3

		uObj, err := v2.NewUnstructured(&obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(uObj.GetType()).To(Equal("example"))

		res := example{}
		Expect(uObj.DecodeInto(&res)).To(Succeed())
		Expect(res).To(Equal(obj))
	})

})
