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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Constructor", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeRegistry      blueprintsregistry.Registry
		fakeCompRepo      componentsregistry.Registry
	)

	BeforeEach(func() {
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
	})

	It("should construct the imported config from a sibling", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstB, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test2/b"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstB
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"b.a": "val-a",
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		res, err := c.Construct(ctx, inInstB)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		Expect(res).To(Equal(expectedConfig))
	})

	It("should construct the imported config from a sibling and the indirect parent import", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstC, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test2/c"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstC
		Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())

		expectedConfig := map[string]interface{}{
			"c.a": "val-a",
			"c.b": "val-root-import", // from root import
		}

		Expect(op.SetInstallationContext(ctx)).To(Succeed())
		c := imports.NewConstructor(op)
		res, err := c.Construct(ctx, inInstC)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())

		Expect(res).To(Equal(expectedConfig))
	})

	Context("schema validation", func() {
		It("should forbid when the import of a component does not satisfy the schema", func() {
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

			do := &lsv1alpha1.DataObject{}
			do.Name = lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromInstallation(inInstRoot.Info), "root.a")
			do.Namespace = inInstRoot.Info.Namespace
			do.Data = []byte("7")
			Expect(fakeClient.Update(ctx, do)).To(Succeed())

			c := imports.NewConstructor(op)
			_, err = c.Construct(ctx, inInstA)
			Expect(err).To(HaveOccurred())
			Expect(installations.IsSchemaValidationFailedError(err)).To(BeTrue())
		})
	})

	Context("Targets", func() {
		It("should construct import from a manually added target", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			res, err := c.Construct(ctx, inInstRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())

			Expect(res).To(HaveKeyWithValue("root.a", MatchKeys(IgnoreExtras, Keys{
				"spec": MatchKeys(IgnoreExtras, Keys{
					"type":   Equal("landscaper.gardener.cloud/mock"),
					"config": Equal("val-e"),
				}),
			})))
		})

		It("should construct import from a parent import", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstF, err := installations.CreateInternalInstallation(ctx, op, fakeInstallations["test4/f"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstF
			Expect(op.ResolveComponentDescriptors(ctx)).To(Succeed())
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := imports.NewConstructor(op)
			res, err := c.Construct(ctx, inInstF)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())

			Expect(res).To(HaveKeyWithValue("f.a", MatchKeys(IgnoreExtras, Keys{
				"spec": MatchKeys(IgnoreExtras, Keys{
					"type":   Equal("landscaper.gardener.cloud/mock"),
					"config": Equal("val-e"),
				}),
			})))
		})
	})

})
