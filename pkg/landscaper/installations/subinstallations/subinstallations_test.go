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

	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
)

var _ = g.Describe("SubInstallation", func() {

	var (
		op               *installations.Operation
		ctrl             *gomock.Controller
		mockClient       *mock_client.MockClient
		mockStatusWriter *mock_client.MockStatusWriter
		fakeRegistry     *fake.FakeRegistry

		once sync.Once
	)

	g.BeforeEach(func() {
		var err error
		ctrl = gomock.NewController(g.GinkgoT())
		mockClient = mock_client.NewMockClient(ctrl)
		mockStatusWriter = mock_client.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)

		once.Do(func() {
			fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
			Expect(err).ToNot(HaveOccurred())
		})

		commonOp := lsoperation.NewOperation(testing.NullLogger{}, mockClient, kubernetes.LandscaperScheme, fakeRegistry)
		op = &installations.Operation{Interface: commonOp}
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

			si := subinstallations.New(op)
			err := si.Ensure(context.TODO(), &lsv1alpha1.Installation{}, &lsv1alpha1.ComponentDefinition{})
			Expect(err).ToNot(HaveOccurred())
		})

		g.It("should create one installation if a definition references is defined", func() {
			var resInst *lsv1alpha1.Installation
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
				},
			)

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			def := &lsv1alpha1.ComponentDefinition{
				DefinitionReferences: []lsv1alpha1.DefinitionReference{
					{
						Name:      "def1",
						Reference: "def1:1.0.0",
						Imports: []lsv1alpha1.DefinitionImportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "a", To: "b"}},
						},
						Exports: []lsv1alpha1.DefinitionExportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "c", To: "d"}},
						},
					},
				},
			}

			si := subinstallations.New(op)
			err := si.Ensure(context.TODO(), inst, def)
			Expect(err).ToNot(HaveOccurred())

			Expect(resInst.Labels).To(HaveKeyWithValue(lsv1alpha1.EncompassedByLabel, "root"))
			Expect(resInst.Spec.DefinitionRef).To(Equal("def1:1.0.0"))
			Expect(resInst.Spec.Imports).To(ContainElement(lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "a", To: "b"},
			}))
			Expect(resInst.Spec.Exports).To(ContainElement(lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "c", To: "d"},
			}))
		})

		g.It("should update the status and add the newly created sub installations", func() {
			var resInst *lsv1alpha1.Installation
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
					resInst.Name = "my-inst"
				},
			)

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			inst.Namespace = "default"
			def := &lsv1alpha1.ComponentDefinition{
				DefinitionReferences: []lsv1alpha1.DefinitionReference{
					{
						Name:      "def1",
						Reference: "def1:1.0.0",
						Imports: []lsv1alpha1.DefinitionImportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "a", To: "b"}},
						},
						Exports: []lsv1alpha1.DefinitionExportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "c", To: "d"}},
						},
					},
				},
			}

			si := subinstallations.New(op)
			err := si.Ensure(context.TODO(), inst, def)
			Expect(err).ToNot(HaveOccurred())

			Expect(inst.Status.InstallationReferences).To(HaveLen(1))
			Expect(inst.Status.InstallationReferences).To(ConsistOf(lsv1alpha1.NamedObjectReference{
				Name: "def1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "my-inst",
					Namespace: "default",
				},
			}))
		})

		g.It("should add undefined imports as mappings of the definition to a new created installation", func() {
			var resInst *lsv1alpha1.Installation
			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					resInst = inst
				},
			)

			inst := &lsv1alpha1.Installation{}
			inst.Name = "root"
			def := &lsv1alpha1.ComponentDefinition{
				DefinitionReferences: []lsv1alpha1.DefinitionReference{
					{
						Name:      "def1",
						Reference: "def1:1.0.0",
						Imports: []lsv1alpha1.DefinitionImportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "a", To: "b"}},
						},
						Exports: []lsv1alpha1.DefinitionExportMapping{
							{DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "c", To: "d"}},
						},
					},
				},
			}

			subdef := lsv1alpha1.ComponentDefinition{
				Name:    "def1",
				Version: "1.0.0",
				Imports: []lsv1alpha1.DefinitionImport{
					{DefinitionFieldValue: lsv1alpha1.DefinitionFieldValue{Key: "b"}},
					{DefinitionFieldValue: lsv1alpha1.DefinitionFieldValue{Key: "y"}},
				},
				Exports: []lsv1alpha1.DefinitionExport{
					{DefinitionFieldValue: lsv1alpha1.DefinitionFieldValue{Key: "c"}},
					{DefinitionFieldValue: lsv1alpha1.DefinitionFieldValue{Key: "z"}},
				},
			}

			op = &installations.Operation{
				Interface: lsoperation.NewOperation(testing.NullLogger{}, mockClient, kubernetes.LandscaperScheme, fake.NewFakeRegistry(fake.DefinitionReference{Definition: subdef})),
			}

			si := subinstallations.New(op)
			err := si.Ensure(context.TODO(), inst, def)
			Expect(err).ToNot(HaveOccurred())

			Expect(resInst.Spec.Imports).To(ContainElement(lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "y", To: "y"},
			}))
			Expect(resInst.Spec.Exports).To(ContainElement(lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{From: "z", To: "z"},
			}))
		})

		g.It("should create multiple installations for all definition references", func() {
			subInstallations := make([]*lsv1alpha1.Installation, 0)

			mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
			mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(
				func(ctx context.Context, inst *lsv1alpha1.Installation) {
					subInstallations = append(subInstallations, inst)
				},
			)

			inst := &lsv1alpha1.Installation{}
			def := &lsv1alpha1.ComponentDefinition{
				DefinitionReferences: []lsv1alpha1.DefinitionReference{
					{
						Name:      "def1",
						Reference: "def1:1.0.0",
						Imports: []lsv1alpha1.DefinitionImportMapping{
							{
								DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
									From: "a",
									To:   "b",
								},
							},
						},
						Exports: []lsv1alpha1.DefinitionExportMapping{
							{
								DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
									From: "c",
									To:   "d",
								},
							},
						},
					},
					{
						Name:      "def2",
						Reference: "def1:1.0.0",
					},
				},
			}

			si := subinstallations.New(op)
			err := si.Ensure(context.TODO(), inst, def)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(subInstallations)).To(Equal(2))
		})
	})

	g.It("should not update a reference if nothing has changed", func() {
		inst := &lsv1alpha1.Installation{
			Status: lsv1alpha1.InstallationStatus{
				InstallationReferences: []lsv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: lsv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
				},
			},
		}
		def := &lsv1alpha1.ComponentDefinition{
			DefinitionReferences: []lsv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
					Imports: []lsv1alpha1.DefinitionImportMapping{
						{
							DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
								From: "a",
								To:   "b",
							},
						},
					},
					Exports: []lsv1alpha1.DefinitionExportMapping{
						{
							DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
								From: "c",
								To:   "d",
							},
						},
					},
				},
			},
		}

		subinst := &lsv1alpha1.Installation{}
		subinst.Name = "inst-def1"
		subinst.Namespace = "default"
		subinst.Spec.DefinitionRef = "def1:1.0.0"
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
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
			func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
				Expect(key.Name).To(Equal("inst-def1"))
				Expect(key.Namespace).To(Equal("default"))
				*obj = *subinst
			},
		)

		si := subinstallations.New(op)
		err := si.Ensure(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	g.It("should reinstall a subinstallation that does not exist anymore", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

		inst := &lsv1alpha1.Installation{
			Status: lsv1alpha1.InstallationStatus{
				InstallationReferences: []lsv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: lsv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
					{
						Name: "def2",
						Reference: lsv1alpha1.ObjectReference{
							Name:      "inst-def2",
							Namespace: "default",
						},
					},
				},
			},
		}
		def := &lsv1alpha1.ComponentDefinition{
			DefinitionReferences: []lsv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
				},
			},
		}

		si := subinstallations.New(op)
		err := si.Ensure(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	g.It("should remove a subinstallation that is not referenced anymore", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		inst := &lsv1alpha1.Installation{}
		inst.Status.InstallationReferences = []lsv1alpha1.NamedObjectReference{
			{
				Name: "def1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "inst-def1",
					Namespace: "default",
				},
			},
		}
		def := &lsv1alpha1.ComponentDefinition{}
		subinst := &lsv1alpha1.Installation{}
		subinst.Name = "inst-def1"

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
			Expect(key.Name).To(Equal("inst-def1"))
			*obj = *subinst
		})
		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		si := subinstallations.New(op)
		err := si.Ensure(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	g.It("should wait until all subinstallations are not in progressing state", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		inst := &lsv1alpha1.Installation{}
		inst.Status.InstallationReferences = []lsv1alpha1.NamedObjectReference{
			{
				Name: "def1",
				Reference: lsv1alpha1.ObjectReference{
					Name:      "inst-def1",
					Namespace: "default",
				},
			},
		}
		def := &lsv1alpha1.ComponentDefinition{
			DefinitionReferences: []lsv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.1.0",
				},
			},
		}
		subinst := &lsv1alpha1.Installation{}
		subinst.Name = "inst-def1"
		subinst.Status.Phase = lsv1alpha1.ComponentPhaseProgressing

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *lsv1alpha1.Installation) {
			Expect(key.Name).To(Equal("inst-def1"))
			*obj = *subinst
		})

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

		si := subinstallations.New(op)
		err := si.Ensure(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

})
