// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/reconcilehelper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("DeployItemExecutions", func() {

	var (
		op                *installations.Operation
		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
	)

	Load := func(inst string) (context.Context, *installations.InstallationImportsAndBlueprint) {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations[inst])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		rh, err := reconcilehelper.NewReconcileHelper(ctx, op)
		Expect(err).ToNot(HaveOccurred())
		imps, err := rh.ImportsSatisfied()
		Expect(err).NotTo(HaveOccurred())
		c := imports.NewConstructor(op)
		Expect(c.Construct(ctx, imps)).To(Succeed())
		return ctx, inInstRoot
	}

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		createDefaultContextsForNamespace(fakeClient)
		fakeInstallations = state.Installations

		registryAccess, err := registries.GetFactory().NewLocalRegistryAccess("./testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registryAccess),
		}
	})

	It("should correctly reference targets in deployitem specifications", func() {
		ctx, inst := Load("test2/root")
		exec := executions.New(op)
		execTemplates, err := exec.RenderDeployItemTemplates(ctx, inst)
		Expect(err).To(Succeed())
		Expect(execTemplates).To(HaveLen(3))
		compareTo := &core.ObjectReference{
			Name:      "mytarget",
			Namespace: "test2",
		}
		Expect(execTemplates[0].Target).To(Equal(compareTo))
		Expect(execTemplates[1].Target).To(Equal(compareTo))
		Expect(execTemplates[2].Target).To(Equal(compareTo))
	})

	It("should fail if targetlist index is out-of-bounds", func() {
		ctx, inst := Load("test2/import-index-wrong")
		exec := executions.New(op)
		_, err := exec.RenderDeployItemTemplates(ctx, inst)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("index 5 out of bounds"))
	})

	It("should fail if target import does not exist", func() {
		ctx, inst := Load("test2/import-not-exist")
		exec := executions.New(op)
		_, err := exec.RenderDeployItemTemplates(ctx, inst)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid deployitem specification \"myDi\": target import \"foo\" not found"))
	})

	It("should fail if target import is accessed with index", func() {
		ctx, inst := Load("test2/import-wrong-type-1")
		exec := executions.New(op)
		_, err := exec.RenderDeployItemTemplates(ctx, inst)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid deployitem specification \"myDi\": targetlist import \"targetImp\" not found"))
	})

	It("should fail if targetlist import is accessed without index", func() {
		ctx, inst := Load("test2/import-wrong-type-2")
		exec := executions.New(op)
		_, err := exec.RenderDeployItemTemplates(ctx, inst)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid deployitem specification \"myDi\": target import \"targetListImp\" not found"))
	})

})
