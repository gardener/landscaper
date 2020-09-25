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

package imports_test

import (
	"context"

	"github.com/go-logr/logr/testing"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = g.Describe("Validation", func() {

	var (
		defaultTestInstallationConfig *utils.TestInstallationConfig
		op                            *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeRegistry      blueprintsregistry.Registry
		fakeCompRepo      componentsregistry.Registry
	)

	g.BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		fakeInstallations = state.Installations

		fakeRegistry, err = blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())
		fakeCompRepo, err = componentsregistry.NewLocalClient(testing.NullLogger{}, "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Interface: lsoperation.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeRegistry, fakeCompRepo),
		}
		defaultTestInstallationConfig = &utils.TestInstallationConfig{
			RemoteBlueprintBaseURL: "../testdata/registry",
		}
	})

	g.Context("root", func() {
		//g.It("should import data from the static config", func() {
		//	defaultTestInstallationConfig.Installation = fakeInstallations["test1/root"]
		//	defaultTestInstallationConfig.BlueprintFilePath = "../testdata/registry/root/blueprint.yaml"
		//	defaultTestInstallationConfig.BlueprintContentPath = "../testdata/registry/root"
		//	_, inInstRoot, _, instOp := utils.CreateTestInstallationResources(op, *defaultTestInstallationConfig)
		//
		//	value, err := yaml.Marshal(map[string]interface{}{
		//		"ext": map[string]interface{}{
		//			"a": "val1",
		//		},
		//	})
		//	Expect(err).ToNot(HaveOccurred())
		//	inInstRoot.Info.Spec.StaticData = []lsv1alpha1.StaticDataSource{{Value: value}}
		//
		//	val := imports.NewValidator(instOp)
		//	Expect(val.Validate(context.TODO(), inInstRoot)).To(Succeed())
		//})

		g.It("should reject the import from static data if the import is of the wrong type", func() {
			defaultTestInstallationConfig.Installation = fakeInstallations["test1/root"]
			defaultTestInstallationConfig.BlueprintContentPath = "../testdata/registry/root"
			_, inInstRoot, _, instOp := utils.CreateTestInstallationResources(op, *defaultTestInstallationConfig)

			_, err := yaml.Marshal(map[string]interface{}{
				"ext": map[string]interface{}{
					"a": true,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			val := imports.NewValidator(instOp)
			Expect(val.Validate(context.TODO(), inInstRoot)).To(HaveOccurred())
		})
	})

	g.It("should successfully validate when the import of a component is defined by its parent", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		op.Context().Parent = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		Expect(val.Validate(ctx, inInstA)).To(Succeed())
	})

	g.It("should successfully validate when the import of a component is defined by a sibling and all sibling dependencies are completed", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		op.Context().Parent = inInstRoot
		op.Context().Siblings = []*installations.Installation{inInstA}
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		Expect(val.Validate(ctx, inInstB)).To(Succeed())
	})

	g.It("should reject the validation when the parent component is not progressing", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())
		inInstRoot.Info.Status.Phase = lsv1alpha1.ComponentPhaseInit
		Expect(fakeClient.Status().Update(ctx, inInstRoot.Info))

		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		err = val.Validate(ctx, inInstA)
		Expect(err).To(HaveOccurred())
		Expect(installations.IsImportNotSatisfiedError(err)).To(BeTrue())
	})

	g.It("should reject when a direct sibling dependency is still running", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())

		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		op.Context().Parent = inInstRoot
		op.Context().Siblings = []*installations.Installation{inInstA}
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		err = val.Validate(ctx, inInstB)
		Expect(err).ToNot(HaveOccurred())
	})

	g.It("should reject when a dependent sibling has not finished yet", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/a"])
		Expect(err).ToNot(HaveOccurred())
		inInstA.Info.Status.Phase = lsv1alpha1.ComponentPhaseProgressing
		Expect(fakeClient.Update(ctx, inInstA.Info)).To(Succeed())

		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())
		inInstB.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstC, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/c"])
		Expect(err).ToNot(HaveOccurred())
		inInstC.Info.Status.Phase = lsv1alpha1.ComponentPhaseSucceeded

		inInstD, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/d"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstD

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Context().Parent = inInstRoot
		op.Context().Siblings = []*installations.Installation{inInstA, inInstB, inInstC}
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		err = val.Validate(ctx, inInstD)
		Expect(err).To(HaveOccurred())
		Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
	})

	g.It("should reject when a dependent sibling of my parent that has not finished yet", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstA, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test3/a"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstA
		Expect(op.ResolveComponentDescriptors(context.TODO())).To(Succeed())

		inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test3/root"])
		Expect(err).ToNot(HaveOccurred())

		op.Context().Parent = inInstRoot
		op.Context().Siblings = []*installations.Installation{inInstA}
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		val := imports.NewValidator(op)
		err = val.Validate(ctx, inInstA)
		Expect(err).To(HaveOccurred())
		Expect(installations.IsNotCompletedDependentsError(err)).To(BeTrue())
	})

})
