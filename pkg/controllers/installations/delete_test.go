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
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	var (
		op lsoperation.Interface

		state        *envtest.State
		fakeRegistry *fake.FakeRegistry
	)

	BeforeEach(func() {
		var err error
		fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(testing.NullLogger{}, testenv.Client, kubernetes.LandscaperScheme, fakeRegistry)
	})

	AfterEach(func() {
		if state != nil {
			ctx := context.Background()
			defer ctx.Done()
			Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
			state = nil
		}
	})

	It("should not delete if another installation still imports a exported value", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(fakeRegistry, state.Installations["a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstA)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instC := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
		Expect(instC.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should block deletion if there are still subinstallations", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())

		inInstRoot, err := installations.CreateInternalInstallation(fakeRegistry, state.Installations["root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstRoot)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instA := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "a", Namespace: state.Namespace}, instA)).ToNot(HaveOccurred())
		instB := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "b", Namespace: state.Namespace}, instB)).ToNot(HaveOccurred())

		Expect(instA.DeletionTimestamp.IsZero()).To(BeFalse())
		Expect(instB.DeletionTimestamp.IsZero()).To(BeFalse())
	})

	It("should not block deletion if there are no subinstallations left", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test1")
		Expect(err).ToNot(HaveOccurred())

		inInstB, err := installations.CreateInternalInstallation(fakeRegistry, state.Installations["b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should delete subinstallations if no one imports exported values", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state/test2")
		Expect(err).ToNot(HaveOccurred())

		inInstB, err := installations.CreateInternalInstallation(fakeRegistry, state.Installations["a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, nil, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instC := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
		Expect(instC.DeletionTimestamp.IsZero()).To(BeFalse())
	})

})
