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

package installations_test

import (
	"context"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	installationsctl "github.com/gardener/landscaper/pkg/controllers/installations"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	"github.com/gardener/landscaper/test/utils/fake_client"
)


var _ = Describe("Delete", func() {

	var (
		op lsoperation.Interface

		fakeInstallations map[string]*lsv1alpha1.ComponentInstallation
		fakeClient        client.Client
		fakeRegistry      *fake.FakeRegistry
	)

	BeforeEach(func() {
		var (
			err   error
			state *fake_client.State
		)
		fakeClient, state, err = fake_client.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())
		fakeInstallations = state.Installations

		fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeRegistry)
	})

	It("should not delete if another installation still imports a exported value", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstA, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstA)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		Expect(fakeInstallations["test1/c"].DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should block deletion if there are still subinstallations", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstRoot)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		Expect(fakeInstallations["test1/a"].DeletionTimestamp.IsZero()).To(BeTrue())
		Expect(fakeInstallations["test1/b"].DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should not block deletion if there are no subinstallations left", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstB, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should delete subinstallations if no one imports exported values", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstB, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		Expect(fakeInstallations["test1/c"].DeletionTimestamp.IsZero()).To(BeFalse())
	})

})