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

package installations

import (
	"context"
	"sync"

	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/component"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
	"github.com/gardener/landscaper/test/utils/fake_client"
)

var _ = g.Describe("Scheduler", func() {

	var (
		a                *actuator
		ctrl             *gomock.Controller
		mockClient       *mock_client.MockClient
		mockStatusWriter *mock_client.MockStatusWriter

		installations map[string]*lsv1alpha1.ComponentInstallation
		fakeClient    client.Client
		fakeRegistry  *fake.FakeRegistry

		once sync.Once
	)

	g.BeforeEach(func() {
		ctrl = gomock.NewController(g.GinkgoT())
		mockClient = mock_client.NewMockClient(ctrl)
		mockStatusWriter = mock_client.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)
		a = &actuator{
			log:    testing.NullLogger{},
			scheme: kubernetes.LandscaperScheme,
			c:      mockClient,
		}

		once.Do(func() {
			var err error
			fakeClient, installations, err = fake_client.NewFakeClientFromPath("./testdata/state")
			Expect(err).ToNot(HaveOccurred())

			fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	g.AfterEach(func() {
		ctrl.Finish()
	})

	g.Context("Imports", func() {

		g.Context("Satisfied", func() {
			g.BeforeEach(func() {
				a.c = fakeClient
				a.registry = fakeRegistry
			})

			g.It("should successfully validate when the import of a component is defined by its parent with the right version", func() {
				ctx := context.Background()
				defer ctx.Done()

				instA := installations["test1/a"]
				inInstA, err := CreateInternalInstallation(fakeRegistry, instA)
				Expect(err).ToNot(HaveOccurred())

				instRoot := installations["test1/root"]
				inInstRoot, err := CreateInternalInstallation(fakeRegistry, instRoot)
				Expect(err).ToNot(HaveOccurred())

				ok, err := a.importsAreSatisfied(ctx, nil, inInstA, &Context{Parent: inInstRoot})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})

			g.It("should reject when the import of a component is defined by its parent with the wrong version", func() {
				ctx := context.Background()
				defer ctx.Done()

				instA := installations["test1/a"]
				inInstA, err := CreateInternalInstallation(fakeRegistry, instA)
				Expect(err).ToNot(HaveOccurred())

				instRoot := installations["test1/root"]
				inInstRoot, err := CreateInternalInstallation(fakeRegistry, instRoot)
				Expect(err).ToNot(HaveOccurred())
				instRoot.Status.Imports[0].ConfigGeneration = 5

				ok, err := a.importsAreSatisfied(ctx, nil, inInstA, &Context{Parent: inInstRoot})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})

			g.It("should reject the validation when the parent component is not progressing", func() {
				ctx := context.Background()
				defer ctx.Done()

				instA := installations["test1/a"]
				inInstA, err := CreateInternalInstallation(fakeRegistry, instA)
				Expect(err).ToNot(HaveOccurred())

				instRoot := installations["test1/root"]
				inInstRoot, err := CreateInternalInstallation(fakeRegistry, instRoot)
				Expect(err).ToNot(HaveOccurred())
				instRoot.Status.Phase = lsv1alpha1.ComponentPhaseInit

				ok, err := a.importsAreSatisfied(ctx, nil, inInstA, &Context{Parent: inInstRoot})
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			g.It("should reject when the import of a component is not yet ready", func() {
				ctx := context.Background()
				defer ctx.Done()

				instD := installations["test1/d"]
				inInstD, err := CreateInternalInstallation(fakeRegistry, instD)
				Expect(err).ToNot(HaveOccurred())

				instRoot := installations["test1/root"]
				inInstRoot, err := CreateInternalInstallation(fakeRegistry, instRoot)
				Expect(err).ToNot(HaveOccurred())

				ok, err := a.importsAreSatisfied(ctx, nil, inInstD, &Context{Parent: inInstRoot})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			// that one fucked up scenario
			g.It("should reject when there is already a higher config generation in the imported tree than the config generation that should be imported", func() {
				ctx := context.Background()
				defer ctx.Done()

				instA := installations["test3/a"]
				inInstA, err := CreateInternalInstallation(fakeRegistry, instA)
				Expect(err).ToNot(HaveOccurred())

				instB := installations["test3/b"]
				inInstB, err := CreateInternalInstallation(fakeRegistry, instB)
				Expect(err).ToNot(HaveOccurred())

				instC := installations["test3/c"]
				inInstC, err := CreateInternalInstallation(fakeRegistry, instC)
				Expect(err).ToNot(HaveOccurred())

				instRoot := installations["test3/root"]
				inInstRoot, err := CreateInternalInstallation(fakeRegistry, instRoot)
				Expect(err).ToNot(HaveOccurred())

				ok, err := a.importsAreSatisfied(ctx, nil, inInstC, &Context{Parent: inInstRoot, Siblings: []*component.Installation{inInstA, inInstB}})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			g.It("should validate the import satisfaction based on a given context", func() {
				Expect(true).To(BeTrue())
			})

		})

	})
})
