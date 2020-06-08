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
	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
	"github.com/gardener/landscaper/test/utils/fake_client"
	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

var _ = Describe("Scheduler", func() {

	var (
		a                *actuator
		ctrl             *gomock.Controller
		mockClient       *mock_client.MockClient
		mockStatusWriter *mock_client.MockStatusWriter
		fakeClient       client.Client

		once sync.Once
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
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
			fakeClient, err = fake_client.NewFakeClientFromPath("./testdata/state")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Context", func() {
		BeforeEach(func() {
			a.c = fakeClient
		})

		It("should show no parent nor siblings for the test1 root", func() {
			ctx := context.Background()
			defer ctx.Done()

			inst := &lsv1alpha1.ComponentInstallation{}
			err := fakeClient.Get(ctx, client.ObjectKey{Name: "root", Namespace: "test1"}, inst)
			Expect(err).ToNot(HaveOccurred())

			parent, siblings, err := a.determineInstallationContext(ctx, inst)
			Expect(err).ToNot(HaveOccurred())

			Expect(parent).To(BeNil())
			// should be 0, but this is currently a workaround until this issue https://github.com/kubernetes-sigs/controller-runtime/issues/866 is fixed
			Expect(siblings).To(HaveLen(1))
		})

		It("should show no parent and one sibling for the test2 a installation", func() {
			ctx := context.Background()
			defer ctx.Done()

			inst := &lsv1alpha1.ComponentInstallation{}
			err := fakeClient.Get(ctx, client.ObjectKey{Name: "a", Namespace: "test2"}, inst)
			Expect(err).ToNot(HaveOccurred())

			parent, siblings, err := a.determineInstallationContext(ctx, inst)
			Expect(err).ToNot(HaveOccurred())

			Expect(parent).To(BeNil())
			// should be 1, but this is currently a workaround until this issue https://github.com/kubernetes-sigs/controller-runtime/issues/866 is fixed
			Expect(siblings).To(HaveLen(2))
			//Expect(siblings[0].Name).To(Equal("b"))
		})

		It("should correctly determine the visible context of a installation with its parent and sibling installations", func() {
			ctx := context.Background()
			defer ctx.Done()

			inst := &lsv1alpha1.ComponentInstallation{}
			err := fakeClient.Get(ctx, client.ObjectKey{Name: "b", Namespace: "test1"}, inst)
			Expect(err).ToNot(HaveOccurred())

			parent, siblings, err := a.determineInstallationContext(ctx, inst)
			Expect(err).ToNot(HaveOccurred())

			Expect(parent).ToNot(BeNil())
			Expect(siblings).To(HaveLen(2))

			Expect(parent.Name).To(Equal("root"))
		})
	})

	Context("Imports", func() {

		Context("Satisfied", func() {

			It("should validate the import satisfaction based on a given context", func() {
				Expect(true).To(BeTrue())
			})

		})

	})
})
