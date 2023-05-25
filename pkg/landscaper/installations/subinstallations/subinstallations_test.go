// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations_test

import (
	"context"

	"github.com/gardener/landscaper/pkg/components/cnudie/oci"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("SubInstallation", func() {

	var (
		op                *lsoperation.Operation
		state             *envtest.State
		fakeClient        client.Client
		fakeInstallations map[string]*lsv1alpha1.Installation

		createSubInstallationsOperation = func(ctx context.Context, installation *lsv1alpha1.Installation) *subinstallations.Operation {
			instRoot, err := installations.CreateInternalInstallationWithContext(ctx, installation, fakeClient, op.ComponentsRegistry())
			Expect(err).ToNot(HaveOccurred())

			repoCtx := &cdv2.OCIRegistryRepository{
				ObjectType: cdv2.ObjectType{
					Type: oci.LocalRepositoryType,
				},
				BaseURL: "./testdata/registry",
			}

			repoCtxRaw, err := json.Marshal(repoCtx)
			Expect(err).ToNot(HaveOccurred())

			lsCtx, err := installations.GetInstallationContext(ctx, fakeClient, installation)
			Expect(err).ToNot(HaveOccurred())
			lsCtx.External.Context.RepositoryContext = &cdv2.UnstructuredTypedObject{
				ObjectType: repoCtx.ObjectType,
				Raw:        repoCtxRaw,
			}

			rootInstOp, err := installations.NewOperationBuilder(instRoot).WithOperation(op).WithContext(lsCtx).Build(ctx)
			Expect(err).ToNot(HaveOccurred())
			return subinstallations.New(rootInstOp)
		}

		expectSubInstallationsSucceed = func(ctx context.Context, namespace, rootInstallation string, expectedSubInstallations ...lsv1alpha1.NamedObjectReference) (*lsv1alpha1.Installation, []*lsv1alpha1.Installation) {
			var err error

			inst := fakeInstallations[namespace+"/"+rootInstallation]
			Expect(inst).ToNot(BeNil())
			si := createSubInstallationsOperation(ctx, inst)
			Expect(si.Ensure(ctx)).To(Succeed())

			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(inst), inst)
			Expect(err).ToNot(HaveOccurred())
			Expect(inst.Status.Conditions).To(HaveLen(1))
			Expect(inst.Status.Conditions[0].Type).To(Equal(lsv1alpha1.EnsureSubInstallationsCondition))
			Expect(inst.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))
			Expect(inst.Status.InstallationReferences).To(HaveLen(len(expectedSubInstallations)))

			subInstallationList := make([]*lsv1alpha1.Installation, 0, len(expectedSubInstallations))

			for i, expectedSubInst := range expectedSubInstallations {
				Expect(inst.Status.InstallationReferences[i].Name).To(Equal(expectedSubInst.Name))
				Expect(inst.Status.InstallationReferences[i].Reference.Name).To(HavePrefix(expectedSubInst.Reference.Name))
				Expect(inst.Status.InstallationReferences[i].Reference.Namespace).To(Equal(expectedSubInst.Reference.Namespace))

				subinst := &lsv1alpha1.Installation{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: inst.Status.InstallationReferences[i].Reference.Name, Namespace: inst.Status.InstallationReferences[i].Reference.Namespace}, subinst)
				Expect(err).ToNot(HaveOccurred())

				subInstallationList = append(subInstallationList, subinst)
			}

			return inst, subInstallationList
		}

		expectSubInstallationsFail = func(ctx context.Context, namespace, rootInstallation string) *lsv1alpha1.Installation {
			inst := fakeInstallations[namespace+"/"+rootInstallation]
			Expect(inst).ToNot(BeNil())
			si := createSubInstallationsOperation(ctx, inst)
			Expect(si.Ensure(ctx)).To(HaveOccurred())
			return inst
		}
	)

	BeforeEach(func() {
		var err error
		ctx := context.Background()
		defer ctx.Done()

		state, err = testenv.InitResources(ctx, "./testdata/state")
		Expect(err).ToNot(HaveOccurred())
		fakeClient = testenv.Client
		fakeInstallations = state.Installations

		Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, "test1", "test2", "test3", "test4", "test5", "test6", "test7", "test8", "test9", "test10", "test11", "test12")).To(Succeed())

		registryAccess, err := registries.NewFactory().NewLocalRegistryAccess("./testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op, err = lsoperation.NewBuilder().
			Client(fakeClient).Scheme(api.LandscaperScheme).
			WithEventRecorder(record.NewFakeRecorder(1024)).
			ComponentRegistry(registryAccess).
			Build(context.Background())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.Background(), state)).To(Succeed())
	})

	Context("Create subinstallations", func() {

		It("should not create any installations if no subinstallation definitions are defined", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test1", "root")
		})

		It("should create one installation if a subinstallation is defined", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, subinsts := expectSubInstallationsSucceed(ctx, "test2", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test2"},
			})

			Expect(subinsts[0].Spec.Context).To(Equal("default"))

			Expect(subinsts[0].Spec.ComponentDescriptor.Reference.ComponentName).To(Equal("example.com/root"))
			Expect(subinsts[0].Spec.ComponentDescriptor.Reference.Version).To(Equal("1.0.0"))

			Expect(subinsts[0].Spec.ComponentDescriptor.Reference.RepositoryContext.Object["baseUrl"]).To(Equal("./testdata/registry"))
			Expect(subinsts[0].Spec.ComponentDescriptor.Reference.RepositoryContext.Object["type"]).To(Equal("ociRegistry"))

			Expect(subinsts[0].Spec.Blueprint.Reference.ResourceName).To(Equal("def-1"))

			Expect(subinsts[0].Spec.Imports.Data).To(HaveLen(1))
			Expect(subinsts[0].Spec.Imports.Data[0].Name).To(Equal("a"))
			Expect(subinsts[0].Spec.Imports.Data[0].DataRef).To(Equal("b"))

			Expect(subinsts[0].Spec.Exports.Data).To(HaveLen(1))
			Expect(subinsts[0].Spec.Exports.Data[0].Name).To(Equal("c"))
			Expect(subinsts[0].Spec.Exports.Data[0].DataRef).To(Equal("d"))
		})

		It("should create one installation if a subinstallationExecution is defined", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test3", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test3"},
			})
		})

		It("should create multiple installations for all definition references", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test4", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test4"},
			}, lsv1alpha1.NamedObjectReference{
				Name: "def-2",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-2",
					Namespace: "test4"},
			})
		})

		It("should create multiple installations for all templates defined by default subinstallations and executions", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test5", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-2",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-2",
					Namespace: "test5"},
			}, lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test5"},
			})
		})
	})

	Context("Update", func() {

		It("should set a installation reference even if nothing has changed to trigger a reconcile", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test6", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test6"},
			})
		})

		It("should update a reference even if nothing has changed to trigger a reconcile", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test7", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test7"},
			})
		})

		It("should reinstall a subinstallation that does not exist anymore", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test8", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test8"},
			})
		})

		It("should install subinstallation that references blueprint in a component reference", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, _ = expectSubInstallationsSucceed(ctx, "test11", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test11"},
			})
		})

		XIt("should not update until all subinstallations are not in progressing state", func() {
			ctx := context.Background()
			defer ctx.Done()

			_ = expectSubInstallationsFail(ctx, "test9", "root")
		})

		It("should inherit context definition", func() {
			ctx := context.Background()
			defer ctx.Done()

			_, subinsts := expectSubInstallationsSucceed(ctx, "test12", "root", lsv1alpha1.NamedObjectReference{
				Name: "def-1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "def-1",
					Namespace: "test12"},
			})

			Expect(subinsts[0].Spec.Context).To(Equal("custom"))
		})

		Context("Cleanup", func() {

			It("should remove a subinstallation that is not referenced anymore", func() {
				ctx := context.Background()
				defer ctx.Done()

				inst := fakeInstallations["test10/root"]
				Expect(inst).ToNot(BeNil())
				si := createSubInstallationsOperation(ctx, inst)
				Expect(si.Ensure(ctx)).To(Succeed())

				err := fakeClient.Get(ctx, client.ObjectKeyFromObject(inst), inst)
				Expect(err).ToNot(HaveOccurred())
				Expect(inst.Status.InstallationReferences).To(HaveLen(1))

				subinst := &lsv1alpha1.Installation{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: inst.Status.InstallationReferences[0].Reference.Name, Namespace: inst.Status.InstallationReferences[0].Reference.Namespace}, subinst)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

	})

})
