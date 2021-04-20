// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	k8smock "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
	"github.com/gardener/landscaper/test/utils"
)

var _ = Describe("SubInstallation", func() {

	var (
		op               lsoperation.Interface
		ctrl             *gomock.Controller
		mockClient       *k8smock.MockClient
		mockStatusWriter *k8smock.MockStatusWriter
		fakeCompRepo     ctf.ComponentResolver

		defaultTestConfig *utils.TestInstallationConfig
	)

	BeforeEach(func() {
		var err error
		ctrl = gomock.NewController(GinkgoT())
		mockClient = k8smock.NewMockClient(ctrl)
		mockStatusWriter = k8smock.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)

		fakeCompRepo, err = componentsregistry.NewLocalClient(testing.NullLogger{}, "./testdata")
		Expect(err).ToNot(HaveOccurred())

		op = lsoperation.NewOperation(testing.NullLogger{}, mockClient, api.LandscaperScheme, fakeCompRepo)

		defaultTestConfig = &utils.TestInstallationConfig{
			MockClient:                   mockClient,
			InstallationName:             "root",
			InstallationNamespace:        "default",
			RemoteBlueprintComponentName: "example.com/root",
			RemoteBlueprintResourceName:  "root",
			RemoteBlueprintVersion:       "1.0.0",
			BlueprintFilePath:            "./testdata/01-root/blueprint-root1.yaml",
			BlueprintContentPath:         "./testdata/01-root",
			RemoteBlueprintBaseURL:       "./testdata",
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Create subinstallations", func() {

		It("should not create any installations if no subinstallation definitions are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil).Times(2) // once for the operation and once in the ensure.
			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					Expect(len(inst.Status.Conditions)).To(Equal(1))
					Expect(inst.Status.Conditions[0].Type).To(Equal(lsv1alpha1.EnsureSubInstallationsCondition))
					Expect(inst.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))
				},
			)

			blue := blueprints.New(&lsv1alpha1.Blueprint{}, memoryfs.New())
			inst, err := installations.New(&lsv1alpha1.Installation{}, blue)
			Expect(err).ToNot(HaveOccurred())
			instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst)
			Expect(err).ToNot(HaveOccurred())

			si := subinstallations.New(instOp)
			Expect(si.Ensure(ctx)).To(Succeed())
		})

		It("should create one installation if a subinstallation is defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			var resInst *lsv1alpha1.Installation
			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
				Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())

			Expect(resInst.Labels).To(HaveKeyWithValue(lsv1alpha1.EncompassedByLabel, "root"))
			Expect(resInst.Spec.ComponentDescriptor.Reference).NotTo(BeNil())
			Expect(resInst.Spec.ComponentDescriptor.Reference).To(Equal(&lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: &cdv2.RepositoryContext{
					Type:    "ociRegistry",
					BaseURL: "./testdata",
				},
				ComponentName: "example.com/root",
				Version:       "1.0.0",
			},
			))

			Expect(resInst.Spec.Blueprint.Reference).NotTo(BeNil())
			Expect(resInst.Spec.Blueprint.Reference).To(Equal(&lsv1alpha1.RemoteBlueprintReference{
				ResourceName: "def1",
			}))
			Expect(resInst.Spec.Imports.Data).To(ContainElement(lsv1alpha1.DataImport{
				Name:    "a",
				DataRef: "b",
			}))
			Expect(resInst.Spec.Exports.Data).To(ContainElement(lsv1alpha1.DataExport{
				Name:    "c",
				DataRef: "d",
			}))
		})

		It("should create one installation if a subinstallationExecution is defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root4.yaml"
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subInstallations := make([]*lsv1alpha1.Installation, 0)

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&lsv1alpha1.Installation{}), gomock.Any()).AnyTimes().Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					subInstallations = append(subInstallations, inst)
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())

			Expect(len(subInstallations)).To(Equal(1))
		})

		It("should update the status and add the newly created sub installations", func() {
			ctx := context.Background()
			defer ctx.Done()
			rootInst, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			var resInst *lsv1alpha1.Installation
			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
				Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
					resInst.Name = "my-inst"
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())

			Expect(rootInst.Status.InstallationReferences).To(HaveLen(1))
			Expect(rootInst.Status.InstallationReferences).To(ConsistOf(lsv1alpha1.NamedObjectReference{
				Name: "def1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "my-inst",
					Namespace: "default",
				},
			}))
		})

		It("should create multiple installations for all definition references", func() {
			ctx := context.Background()
			defer ctx.Done()
			defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root2.yaml"
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subInstallations := make([]*lsv1alpha1.Installation, 0)

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					subInstallations = append(subInstallations, inst)
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())

			Expect(len(subInstallations)).To(Equal(2))
		})

		It("should create multiple installations for all templates defined by default subinstallations and executions", func() {
			ctx := context.Background()
			defer ctx.Done()
			defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root5.yaml"
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subInstallations := make([]*lsv1alpha1.Installation, 0)

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					subInstallations = append(subInstallations, inst)
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())

			Expect(len(subInstallations)).To(Equal(2))
		})
	})

	Context("#Update", func() {

		It("should update a reference even if nothing has changed to trigger a reconcile", func() {
			ctx := context.Background()
			defer ctx.Done()
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subinst := &lsv1alpha1.Installation{}
			subinst.Name = "inst-def1"
			subinst.Namespace = "default"
			subinst.Annotations = map[string]string{
				lsv1alpha1.SubinstallationNameAnnotation: "def1",
			}
			subinst.Spec.ComponentDescriptor = utils.LocalRemoteComponentDescriptorRef("root", "1.0.0", ".testdata")
			subinst.Spec.Blueprint = utils.LocalRemoteBlueprintRef("def1")
			subinst.Spec.Imports = lsv1alpha1.InstallationImports{
				Data: []lsv1alpha1.DataImport{
					{
						Name:    "b",
						DataRef: "a",
					},
				},
			}
			subinst.Spec.Exports = lsv1alpha1.InstallationExports{
				Data: []lsv1alpha1.DataExport{
					{
						Name:    "c",
						DataRef: "d",
					},
				},
			}

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil).Do(
				func(ctx context.Context, installations *lsv1alpha1.InstallationList, opts ...client.ListOption) {
					installations.Items = []lsv1alpha1.Installation{
						*subinst,
					}
				},
			)
			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Times(0)
			mockClient.EXPECT().Update(ctx, gomock.Any()).Times(1)
			mockClient.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Times(1).Return(nil).Do( // expect once times for the create and update call
				func(ctx context.Context, key types.NamespacedName, obj *lsv1alpha1.Installation) {
					Expect(key.Name).To(Equal("inst-def1"))
					Expect(key.Namespace).To(Equal("default"))
					*obj = *subinst
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())
		})

		It("should update a reference even if nothing has changed to trigger a reconcile with a fallback to the old status based referencing", func() {
			ctx := context.Background()
			defer ctx.Done()
			rootInst, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			rootInst.Status = lsv1alpha1.InstallationStatus{
				InstallationReferences: []lsv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: lsv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
				},
			}

			subinst := &lsv1alpha1.Installation{}
			subinst.Name = "inst-def1"
			subinst.Namespace = "default"
			subinst.Spec.ComponentDescriptor = utils.LocalRemoteComponentDescriptorRef("root", "1.0.0", ".testdata")
			subinst.Spec.Blueprint = utils.LocalRemoteBlueprintRef("def1")
			subinst.Spec.Imports = lsv1alpha1.InstallationImports{
				Data: []lsv1alpha1.DataImport{
					{
						Name:    "b",
						DataRef: "a",
					},
				},
			}
			subinst.Spec.Exports = lsv1alpha1.InstallationExports{
				Data: []lsv1alpha1.DataExport{
					{
						Name:    "c",
						DataRef: "d",
					},
				},
			}

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil).Do(
				func(ctx context.Context, installations *lsv1alpha1.InstallationList, opts ...client.ListOption) {
					installations.Items = []lsv1alpha1.Installation{
						*subinst,
					}
				},
			)
			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Times(0)
			mockClient.EXPECT().Update(ctx, gomock.Any()).Times(1)
			mockClient.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Times(1).Return(nil).Do( // expect 2 times for the create and update call
				func(ctx context.Context, key types.NamespacedName, obj *lsv1alpha1.Installation) {
					Expect(key.Name).To(Equal("inst-def1"))
					Expect(key.Namespace).To(Equal("default"))
					*obj = *subinst
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())
		})

		It("should reinstall a subinstallation that does not exist anymore", func() {
			ctx := context.Background()
			defer ctx.Done()
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil)
			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Times(1).Return(nil)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(Succeed())
		})

		It("should not update until all subinstallations are not in progressing state", func() {
			ctx := context.Background()
			defer ctx.Done()
			_, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subinst := &lsv1alpha1.Installation{}
			subinst.Name = "inst-def1"
			subinst.Annotations = map[string]string{
				lsv1alpha1.SubinstallationNameAnnotation: "def1",
			}
			subinst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

			mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
				Return(nil).Do(
				func(ctx context.Context, installations *lsv1alpha1.InstallationList, opts ...client.ListOption) {
					installations.Items = []lsv1alpha1.Installation{
						*subinst,
					}
				},
			)
			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Times(0)

			mockClient.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
				Expect(key.Name).To(Equal("inst-def1"))
				*obj = *subinst
			})

			mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).Times(0)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(ctx)).To(HaveOccurred(), "should throw a unable to update call")
		})

		Context("Cleanup", func() {

			It("should remove a subinstallation that is not referenced anymore", func() {
				ctx := context.Background()
				defer ctx.Done()
				defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root3.yaml"
				rootInst, _, _, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

				subinst := &lsv1alpha1.Installation{}
				subinst.Name = "inst-def1"
				subinst.Annotations = map[string]string{
					lsv1alpha1.SubinstallationNameAnnotation: "def1",
				}
				mockClient.EXPECT().List(ctx, gomock.AssignableToTypeOf(&lsv1alpha1.InstallationList{}), gomock.Any()).
					Return(nil).Do(
					func(ctx context.Context, installations *lsv1alpha1.InstallationList, opts ...client.ListOption) {
						installations.Items = []lsv1alpha1.Installation{
							*subinst,
						}
					},
				)
				mockStatusWriter.EXPECT().Update(ctx, gomock.Any()).AnyTimes().Return(nil)
				mockClient.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Times(0)

				mockClient.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
					Expect(key.Name).To(Equal("inst-def1"))
					*obj = *subinst
				})
				mockClient.EXPECT().Delete(ctx, gomock.Any()).Times(1).Return(nil)

				si := subinstallations.New(rootInstOp)
				Expect(si.Ensure(ctx)).To(Succeed())
				Expect(rootInst.Status.InstallationReferences).To(HaveLen(1), "should remove the status for a installation only if the installation is really deleted")
			})
		})
	})

})
