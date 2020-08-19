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

package subinstallations_test

import (
	"context"
	"sync"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprint"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	regapi "github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils/componentrepository"
	k8smock "github.com/gardener/landscaper/pkg/utils/kubernetes/mock"
	"github.com/gardener/landscaper/test/utils"
)

var _ = g.Describe("SubInstallation", func() {

	var (
		op               *installations.Operation
		ctrl             *gomock.Controller
		mockClient       *k8smock.MockClient
		mockStatusWriter *k8smock.MockStatusWriter
		fakeRegistry     regapi.Registry
		fakeCompRepo     componentrepository.Client

		defaultTestConfig *utils.TestInstallationConfig

		once sync.Once
	)

	g.BeforeEach(func() {
		var err error
		ctrl = gomock.NewController(g.GinkgoT())
		mockClient = k8smock.NewMockClient(ctrl)
		mockStatusWriter = k8smock.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)

		once.Do(func() {
			fakeRegistry, err = regapi.NewLocalRegistry(testing.NullLogger{}, "./testdata")
			Expect(err).ToNot(HaveOccurred())

			fakeCompRepo, err = componentrepository.NewLocalClient(testing.NullLogger{}, "./testdata")
			Expect(err).ToNot(HaveOccurred())
		})

		commonOp := lsoperation.NewOperation(testing.NullLogger{}, mockClient, kubernetes.LandscaperScheme, fakeRegistry, fakeCompRepo)
		op = &installations.Operation{Interface: commonOp}

		defaultTestConfig = &utils.TestInstallationConfig{
			MockClient:                   mockClient,
			InstallationName:             "root",
			InstallationNamespace:        "default",
			RemoteBlueprintComponentName: "root",
			RemoteBlueprintResourceName:  "root",
			RemoteBlueprintVersion:       "1.0.0",
			BlueprintFilePath:            "./testdata/01-root/blueprint-root1.yaml",
			BlueprintContentPath:         "./testdata/01-root",
		}
	})

	g.AfterEach(func() {
		ctrl.Finish()
	})

	g.Context("Create subinstallations", func() {

		g.It("should not create any installations if no definition references are defined", func() {
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					Expect(len(inst.Status.Conditions)).To(Equal(1))
					Expect(inst.Status.Conditions[0].Type).To(Equal(lsv1alpha1.EnsureSubInstallationsCondition))
					Expect(inst.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))
				},
			)

			blue, err := blueprint.New(&lsv1alpha1.Blueprint{}, afero.NewMemMapFs())
			Expect(err).ToNot(HaveOccurred())

			si := subinstallations.New(op)
			err = si.Ensure(context.TODO(), &lsv1alpha1.Installation{}, blue)
			Expect(err).ToNot(HaveOccurred())
		})

		g.It("should create one installation if a definition references is defined", func() {
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
			Expect(resInst.Spec.BlueprintRef).To(Equal(lsv1alpha1.RemoteBlueprintReference{
				RepositoryContext: cdv2.RepositoryContext{
					Type: "local",
				},
				VersionedResourceReference: lsv1alpha1.VersionedResourceReference{
					ResourceReference: lsv1alpha1.ResourceReference{
						ComponentName: "root",
						Kind:          "localResource",
						Resource:      "def1",
					},
					Version: "1.0.0",
				},
			}))
			Expect(resInst.Spec.Imports).To(ContainElement(lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "a", To: "b"},
			}))
			Expect(resInst.Spec.Exports).To(ContainElement(lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "c", To: "d"},
			}))
		})

		g.It("should update the status and add the newly created sub installations", func() {
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

		g.It("should add undefined imports as mappings of the definition to a new created installation", func() {
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

			Expect(resInst.Spec.Imports).To(ContainElement(lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "y", To: "y"},
			}))
			Expect(resInst.Spec.Exports).To(ContainElement(lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "z", To: "z"},
			}))
		})

		g.It("should create multiple installations for all definition references", func() {
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

	g.Context("#Update", func() {

		g.It("should update a reference even if nothing has changed to trigger a reconcile", func() {
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
			subinst.Spec.BlueprintRef = utils.LocalRemoteBlueprintRef("root", "def1", "1.0.0")
			subinst.Spec.Imports = []lsv1alpha1.DefinitionImportMapping{
				{
					DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
						From: "a",
						To:   "b",
					},
				},
			}
			subinst.Spec.Exports = []lsv1alpha1.DefinitionExportMapping{
				{
					DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
						From: "c",
						To:   "d",
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

		g.It("should reinstall a subinstallation that does not exist anymore", func() {
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

		g.It("should not update until all subinstallations are not in progressing state", func() {
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

		g.Context("Cleanup", func() {

			g.It("should remove a subinstallation that is not referenced anymore", func() {
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
