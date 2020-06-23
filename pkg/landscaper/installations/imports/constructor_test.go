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

package imports_test

import (
	"context"
	"sync"

	"github.com/go-logr/logr/testing"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/datatype"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	"github.com/gardener/landscaper/test/utils/fake_client"
)

var _ = g.Describe("Validation", func() {

	var (
		op installations.Operation

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

		op = installations.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeRegistry, internalDataTypes)
	})

	g.It("should directly construct the data from the landscape config", func() {
		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		lsConfig, err := landscapeconfig.New(
			&lsv1alpha1.LandscapeConfiguration{
				Status: lsv1alpha1.LandscapeConfigurationStatus{
					ConfigGeneration: 8,
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					lsv1alpha1.DataObjectSecretDataKey: []byte(`{ "ext": { "a": "val1" } }`), // ext.a
				},
			},
		)
		Expect(err).ToNot(HaveOccurred())
		lsConfig.Info.Name = "ls"
		lsConfig.Info.Namespace = "default"

		expectedConfig := map[string]interface{}{
			"root": map[string]interface{}{
				"a": "val1",
			},
		}

		c := imports.NewConstructor(op, lsConfig, nil)
		data, err := c.Construct(context.TODO(), inInstRoot)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).ToNot(BeNil())

		res := make(map[string]interface{})
		Expect(yaml.Unmarshal(data, &res)).ToNot(HaveOccurred())

		Expect(res).To(Equal(expectedConfig))
		Expect(inInstRoot.ImportStatus().GetStates()).To(ConsistOf(lsv1alpha1.ImportState{
			From: "ext.a",
			To:   "root.a",
			SourceRef: &lsv1alpha1.TypedObjectReference{
				APIGroup: "landscaper.gardener.cloud/v1alpha1",
				Kind:     "LandscapeConfiguration",
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      "ls",
					Namespace: "default",
				},
			},
			ConfigGeneration: 8,
		}))
	})

	g.It("should construct the imported config from a sibling", func() {
		inInstA, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		inInstB, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/b"])
		Expect(err).ToNot(HaveOccurred())

		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/root"])
		Expect(err).ToNot(HaveOccurred())

		expectedConfig := map[string]interface{}{
			"b": map[string]interface{}{
				"a": "val-a",
			},
		}

		c := imports.NewConstructor(op, nil, inInstRoot, inInstA)
		data, err := c.Construct(context.TODO(), inInstB)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).ToNot(BeNil())

		res := make(map[string]interface{})
		Expect(yaml.Unmarshal(data, &res)).ToNot(HaveOccurred())

		Expect(res).To(Equal(expectedConfig))
	})

	g.It("should construct the imported config from a sibling and the indirect parent import", func() {
		inInstA, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		inInstC, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/c"])
		Expect(err).ToNot(HaveOccurred())

		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/root"])
		Expect(err).ToNot(HaveOccurred())

		lsConfig, err := landscapeconfig.New(
			&lsv1alpha1.LandscapeConfiguration{
				Status: lsv1alpha1.LandscapeConfigurationStatus{
					ConfigGeneration: 8,
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					lsv1alpha1.DataObjectSecretDataKey: []byte(`{ "ext": { "a": "val1" } }`), // ext.a
				},
			},
		)
		Expect(err).ToNot(HaveOccurred())

		expectedConfig := map[string]interface{}{
			"c": map[string]interface{}{
				"a": "val-a",
				"b": "val1",
			},
		}

		c := imports.NewConstructor(op, lsConfig, inInstRoot, inInstA)
		data, err := c.Construct(context.TODO(), inInstC)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).ToNot(BeNil())

		res := make(map[string]interface{})
		Expect(yaml.Unmarshal(data, &res)).ToNot(HaveOccurred())

		Expect(res).To(Equal(expectedConfig))
	})

})
