// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	installationsctl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Delete", func() {

	var (
		op lsoperation.Interface

		state        *envtest.State
		fakeCompRepo ctf.ComponentResolver
	)

	BeforeEach(func() {
		var err error
		fakeCompRepo, err = componentsregistry.NewLocalClient(testing.NullLogger{}, "./testdata")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(testing.NullLogger{}, testenv.Client, api.LandscaperScheme, fakeCompRepo)
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

		inInstA, err := installations.CreateInternalInstallation(context.TODO(), op, state.Installations[state.Namespace+"/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstA)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(installationsctl.SiblingImportError))

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

		inInstRoot, err := installations.CreateInternalInstallation(context.TODO(), op, state.Installations[state.Namespace+"/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstRoot)
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

		inInstB, err := installations.CreateInternalInstallation(context.TODO(), op, state.Installations[state.Namespace+"/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB)
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

		inInstB, err := installations.CreateInternalInstallation(context.TODO(), op, state.Installations[state.Namespace+"/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inInstB)
		Expect(err).ToNot(HaveOccurred())

		err = installationsctl.EnsureDeletion(ctx, instOp)
		Expect(err).To(HaveOccurred())

		instC := &lsv1alpha1.Installation{}
		Expect(testenv.Client.Get(ctx, client.ObjectKey{Name: "c", Namespace: state.Namespace}, instC)).ToNot(HaveOccurred())
		Expect(instC.DeletionTimestamp.IsZero()).To(BeFalse())
	})

})
