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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Context", func() {

	var (
		op lsoperation.Interface

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeCompRepo      ctf.ComponentResolver
	)

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())
		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(testing.NullLogger{}, "./testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeCompRepo)
	})

	It("should show no parent nor siblings for the test1 root", func() {
		ctx := context.Background()
		defer ctx.Done()

		instRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, instRoot)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(0))
	})

	It("should show no parent and one sibling for the test2/a installation", func() {
		ctx := context.Background()
		defer ctx.Done()

		inst, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(1))
		//Expect(siblings[0].Name).To(Equal("b"))
	})

	It("should correctly determine the visible context of a installation with its parent and sibling installations", func() {
		ctx := context.Background()
		defer ctx.Done()

		inst, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).ToNot(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(3))

		Expect(lCtx.Parent.Info.Name).To(Equal("root"))
	})

})
