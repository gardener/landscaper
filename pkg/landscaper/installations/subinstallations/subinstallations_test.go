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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
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
		op               *installations.Operation
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

		commonOp := lsoperation.NewOperation(testing.NullLogger{}, mockClient, kubernetes.LandscaperScheme, fakeCompRepo)
		op = &installations.Operation{Interface: commonOp}

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

		It("should not create any installations if no definition references are defined", func() {
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					Expect(len(inst.Status.Conditions)).To(Equal(1))
					Expect(inst.Status.Conditions[0].Type).To(Equal(lsv1alpha1.EnsureSubInstallationsCondition))
					Expect(inst.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))
				},
			)

			blue, err := blueprints.New(&lsv1alpha1.Blueprint{}, memoryfs.New())
			Expect(err).ToNot(HaveOccurred())

			si := subinstallations.New(op)
			err = si.Ensure(context.TODO(), &lsv1alpha1.Installation{}, blue)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create one installation if a subinstallation is defined", func() {
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			var resInst *lsv1alpha1.Installation
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())

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

		It("should update the status and add the newly created sub installations", func() {
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			var resInst *lsv1alpha1.Installation
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
					resInst.Name = "my-inst"
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())

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
			defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root2.yaml"
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

			subInstallations := make([]*lsv1alpha1.Installation, 0)

			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					subInstallations = append(subInstallations, inst)
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())

			Expect(len(subInstallations)).To(Equal(2))
		})
	})

	Context("#ApplyUpdate", func() {

		It("should update a reference even if nothing has changed to trigger a reconcile", func() {
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

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

			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil).Do( // expect 2 times for the create and update call
				func(ctx context.Context, key types.NamespacedName, obj *lsv1alpha1.Installation) {
					Expect(key.Name).To(Equal("inst-def1"))
					Expect(key.Namespace).To(Equal("default"))
					*obj = *subinst
				},
			)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())
		})

		It("should reinstall a subinstallation that does not exist anymore", func() {
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)

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

			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())
		})

		It("should not update until all subinstallations are not in progressing state", func() {
			rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)
			rootInst.Status.InstallationReferences = []lsv1alpha1.NamedObjectReference{
				{
					Name: "def1",
					Reference: lsv1alpha1.ObjectReference{
						Name:      "inst-def1",
						Namespace: "default",
					},
				},
			}
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			subinst := &lsv1alpha1.Installation{}
			subinst.Name = "inst-def1"
			subinst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
				Expect(key.Name).To(Equal("inst-def1"))
				*obj = *subinst
			})

			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			si := subinstallations.New(rootInstOp)
			Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(HaveOccurred(), "should throw a unable to update call")
		})

		Context("Cleanup", func() {

			It("should remove a subinstallation that is not referenced anymore", func() {
				defaultTestConfig.BlueprintFilePath = "./testdata/01-root/blueprint-root3.yaml"
				rootInst, _, rootBlueprint, rootInstOp := utils.CreateTestInstallationResources(op, *defaultTestConfig)
				rootInst.Status.InstallationReferences = []lsv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: lsv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
				}
				mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

				subinst := &lsv1alpha1.Installation{}
				subinst.Name = "inst-def1"

				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
					Expect(key.Name).To(Equal("inst-def1"))
					*obj = *subinst
				})
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1).Return(nil)

				si := subinstallations.New(rootInstOp)
				Expect(si.Ensure(context.TODO(), rootInst, rootBlueprint)).To(Succeed())
				Expect(rootInst.Status.InstallationReferences).To(HaveLen(1), "should remove the status for a installation only if the installation is really deleted")
			})
		})
	})

})
