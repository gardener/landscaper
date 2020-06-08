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

package installations

import (
	"context"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	landscaperv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
)

var _ = Describe("SubInstallation", func() {

	var (
		a                *actuator
		ctrl             *gomock.Controller
		mockClient       *mock_client.MockClient
		mockStatusWriter *mock_client.MockStatusWriter
	)

	BeforeEach(func() {
		a = &actuator{
			log:    testing.NullLogger{},
			scheme: kubernetes.LandscaperScheme,
		}
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mock_client.NewMockClient(ctrl)
		mockStatusWriter = mock_client.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)
		_ = a.InjectClient(mockClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should not create any installations if no definition references are defined", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
			func(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation) {
				Expect(len(inst.Status.Conditions)).To(Equal(1))
				Expect(inst.Status.Conditions[0].Type).To(Equal(landscaperv1alpha1.EnsureSubInstallationsCondition))
				Expect(inst.Status.Conditions[0].Status).To(Equal(landscaperv1alpha1.ConditionTrue))
			},
		)

		err := a.EnsureSubInstallations(context.TODO(), &landscaperv1alpha1.ComponentInstallation{}, &landscaperv1alpha1.ComponentDefinition{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should create one installation if a definition references is defined", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
			func(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation) {
				Expect(inst.Labels).To(HaveKeyWithValue(landscaperv1alpha1.EncompassedByLabel, "root"))
				Expect(inst.Spec.DefinitionRef).To(Equal("def1:1.0.0"))
			},
		)

		inst := &landscaperv1alpha1.ComponentInstallation{}
		inst.Name = "root"
		def := &landscaperv1alpha1.ComponentDefinition{
			DefinitionReferences: []landscaperv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
					Imports: []landscaperv1alpha1.DefinitionImportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
								From: "a",
								To:   "b",
							},
						},
					},
					Exports: []landscaperv1alpha1.DefinitionExportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
								From: "c",
								To:   "d",
							},
						},
					},
				},
			},
		}

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should create multiple installations for all definition references", func() {
		subInstallations := make([]*landscaperv1alpha1.ComponentInstallation, 0)

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(
			func(ctx context.Context, inst *landscaperv1alpha1.ComponentInstallation) {
				subInstallations = append(subInstallations, inst)
			},
		)

		inst := &landscaperv1alpha1.ComponentInstallation{}
		def := &landscaperv1alpha1.ComponentDefinition{
			DefinitionReferences: []landscaperv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
					Imports: []landscaperv1alpha1.DefinitionImportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
								From: "a",
								To:   "b",
							},
						},
					},
					Exports: []landscaperv1alpha1.DefinitionExportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
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

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())

		Expect(len(subInstallations)).To(Equal(2))
	})

	It("should not update a reference if nothing has changed", func() {
		inst := &landscaperv1alpha1.ComponentInstallation{
			Status: landscaperv1alpha1.ComponentInstallationStatus{
				InstallationReferences: []landscaperv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: landscaperv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
				},
			},
		}
		def := &landscaperv1alpha1.ComponentDefinition{
			DefinitionReferences: []landscaperv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
					Imports: []landscaperv1alpha1.DefinitionImportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
								From: "a",
								To:   "b",
							},
						},
					},
					Exports: []landscaperv1alpha1.DefinitionExportMapping{
						{
							DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
								From: "c",
								To:   "d",
							},
						},
					},
				},
			},
		}

		subinst := &landscaperv1alpha1.ComponentInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "inst-def1",
				Namespace: "default",
			},
			Spec: landscaperv1alpha1.ComponentInstallationSpec{
				DefinitionRef: "def1:1.0.0",
				Imports: []landscaperv1alpha1.DefinitionImportMapping{
					{
						DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
							From: "a",
							To:   "b",
						},
					},
				},
				Exports: []landscaperv1alpha1.DefinitionExportMapping{
					{
						DefinitionFieldMapping: landscaperv1alpha1.DefinitionFieldMapping{
							From: "c",
							To:   "d",
						},
					},
				},
			},
		}

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil).Do(
			func(ctx context.Context, key client.ObjectKey, obj *landscaperv1alpha1.ComponentInstallation) {
				Expect(key.Name).To(Equal("inst-def1"))
				Expect(key.Namespace).To(Equal("default"))
				*obj = *subinst
			},
		)

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reinstall a subinstallation that does not exist anymore", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

		inst := &landscaperv1alpha1.ComponentInstallation{
			Status: landscaperv1alpha1.ComponentInstallationStatus{
				InstallationReferences: []landscaperv1alpha1.NamedObjectReference{
					{
						Name: "def1",
						Reference: landscaperv1alpha1.ObjectReference{
							Name:      "inst-def1",
							Namespace: "default",
						},
					},
					{
						Name: "def2",
						Reference: landscaperv1alpha1.ObjectReference{
							Name:      "inst-def2",
							Namespace: "default",
						},
					},
				},
			},
		}
		def := &landscaperv1alpha1.ComponentDefinition{
			DefinitionReferences: []landscaperv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.0.0",
				},
			},
		}

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should remove a subinstallation that is not referenced anymore", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		inst := &landscaperv1alpha1.ComponentInstallation{}
		inst.Status.InstallationReferences = []landscaperv1alpha1.NamedObjectReference{
			{
				Name: "def1",
				Reference: landscaperv1alpha1.ObjectReference{
					Name:      "inst-def1",
					Namespace: "default",
				},
			},
		}
		def := &landscaperv1alpha1.ComponentDefinition{}
		subinst := &landscaperv1alpha1.ComponentInstallation{}
		subinst.Name = "inst-def1"

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *landscaperv1alpha1.ComponentInstallation) {
			Expect(key.Name).To(Equal("inst-def1"))
			*obj = *subinst
		})
		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should wait until all subinstallations are not in progressing state", func() {
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		inst := &landscaperv1alpha1.ComponentInstallation{}
		inst.Status.InstallationReferences = []landscaperv1alpha1.NamedObjectReference{
			{
				Name: "def1",
				Reference: landscaperv1alpha1.ObjectReference{
					Name:      "inst-def1",
					Namespace: "default",
				},
			},
		}
		def := &landscaperv1alpha1.ComponentDefinition{
			DefinitionReferences: []landscaperv1alpha1.DefinitionReference{
				{
					Name:      "def1",
					Reference: "def1:1.1.0",
				},
			},
		}
		subinst := &landscaperv1alpha1.ComponentInstallation{}
		subinst.Name = "inst-def1"
		subinst.Status.Phase = landscaperv1alpha1.ComponentPhaseProgressing

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil).Do(func(ctx context.Context, key client.ObjectKey, obj *landscaperv1alpha1.ComponentInstallation) {
			Expect(key.Name).To(Equal("inst-def1"))
			*obj = *subinst
		})

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

		err := a.EnsureSubInstallations(context.TODO(), inst, def)
		Expect(err).ToNot(HaveOccurred())
	})

})
