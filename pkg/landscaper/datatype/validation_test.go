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

package datatype_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	landscaperv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
)

const exampleDirPath = "./testdata/openapiv3"

var _ = Describe("Validation", func() {

	DescribeTable("OpenAPIV3Validations",
		func(scheme landscaperv1alpha1.OpenAPIV3Schema, test Test) {
			err := datatype.Validate(scheme, test.Data)
			if test.Result {
				Expect(err).ToNot(HaveOccurred(), "%v should have been correct", test.Data)
			} else {
				Expect(err).To(HaveOccurred(), "%v should have been incorrect", test.Data)
			}
		}, generateTests()...)
})

func generateTests() []TableEntry {

	entries := []TableEntry{}

	suites, err := readTests()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	for _, suite := range suites {
		for i, test := range suite.Tests {
			entries = append(entries, Entry(
				fmt.Sprintf("should validate test %d of testsuite %s", i, suite.Name),
				suite.OpenAPIV3Schema,
				test,
			))
		}
	}

	return entries
}

func readTests() ([]TestSuite, error) {
	suites := []TestSuite{}
	files, err := ioutil.ReadDir(exampleDirPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(exampleDirPath, file.Name()))
		if err != nil {
			return nil, err
		}

		suite := &TestSuite{}
		err = yaml.Unmarshal(data, suite)
		if err != nil {
			return nil, err
		}

		suites = append(suites, *suite)
		continue
	}
	return suites, nil
}

type TestSuite struct {
	Name            string                             `json:"name"`
	OpenAPIV3Schema landscaperv1alpha1.OpenAPIV3Schema `json:"openAPIV3Schema"`
	Tests           []Test                             `json:"tests"`
}

type Test struct {
	Data   interface{} `json:"data"`
	Result bool        `json:"result"`
}
