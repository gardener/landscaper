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

package exports_test

import (
	"context"
	"sync"

	"github.com/go-logr/logr/testing"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	"github.com/gardener/landscaper/test/utils/fake_client"
)

var _ = g.Describe("Constructor", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.ComponentInstallation
		fakeDataTypes     map[string]*lsv1alpha1.DataType
		fakeClient        client.Client
		fakeRegistry      *fake.FakeRegistry

		once sync.Once
	)

	g.BeforeEach(func() {
		once.Do(func() {
			var (
				err   error
				state *fake_client.State
			)
			fakeClient, state, err = fake_client.NewFakeClientFromPath("./testdata/state")
			Expect(err).ToNot(HaveOccurred())

			fakeInstallations = state.Installations
			fakeDataTypes = state.DataTypes

			fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
			Expect(err).ToNot(HaveOccurred())
		})

		dtArr := make([]lsv1alpha1.DataType, 0)
		for _, dt := range fakeDataTypes {
			dtArr = append(dtArr, *dt)
		}
		internalDataTypes, err := datatype.CreateDatatypesMap(dtArr)
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Interface: lsoperation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeRegistry),
			Datatypes: internalDataTypes,
		}
	})

	g.It("should construct the exported config from its execution", func() {
		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/root"])
		Expect(err).ToNot(HaveOccurred())

		expectedConfig := map[string]interface{}{
			"root": map[string]interface{}{
				"z": "val-exec",
			},
		}

		c := exports.NewConstructor(op)
		res, err := c.Construct(context.TODO(), inInstRoot)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		Expect(res).To(Equal(expectedConfig))
	})

	g.It("should construct the exported config from a siblings", func() {
		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		expectedConfig := map[string]interface{}{
			"root": map[string]interface{}{
				"z": "val-b",
				"y": "val-c",
			},
		}

		c := exports.NewConstructor(op)
		res, err := c.Construct(context.TODO(), inInstRoot)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		Expect(res).To(Equal(expectedConfig))
	})

	g.It("should construct the exported config from a siblings and the execution config", func() {
		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test3/root"])
		Expect(err).ToNot(HaveOccurred())

		expectedConfig := map[string]interface{}{
			"root": map[string]interface{}{
				"z": "val-exec",
				"x": "val-a",
			},
		}

		c := exports.NewConstructor(op)
		res, err := c.Construct(context.TODO(), inInstRoot)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		Expect(res).To(Equal(expectedConfig))
	})

})
