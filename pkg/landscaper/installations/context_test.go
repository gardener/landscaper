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

package installations_test

import (
	"context"
	"sync"

	"github.com/go-logr/logr/testing"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/registry/fake"
	"github.com/gardener/landscaper/test/utils/fake_client"
)

var _ = g.Describe("Context", func() {

	var (
		op installations.Operation

		fakeInstallations map[string]*lsv1alpha1.ComponentInstallation
		fakeClient        client.Client
		fakeRegistry      *fake.FakeRegistry

		once sync.Once
	)

	g.BeforeEach(func() {
		once.Do(func() {
			var err error
			fakeClient, fakeInstallations, err = fake_client.NewFakeClientFromPath("./testdata/state")
			Expect(err).ToNot(HaveOccurred())

			fakeRegistry, err = fake.NewFakeRegistryFromPath("./testdata/registry")
			Expect(err).ToNot(HaveOccurred())
		})

		op = installations.NewOperation(testing.NullLogger{}, fakeClient, kubernetes.LandscaperScheme, fakeRegistry)
	})

	g.It("should show no parent nor siblings for the test1 root", func() {
		ctx := context.Background()
		defer ctx.Done()

		instRoot, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperation(ctx, op, instRoot)
		Expect(err).ToNot(HaveOccurred())
		lCtx, err := instOp.DetermineContext(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(lCtx.Parent).To(BeNil())
		// should be 0, but this is currently a workaround until this issue https://github.com/kubernetes-sigs/controller-runtime/issues/866 is fixed
		Expect(lCtx.Siblings).To(HaveLen(1))
	})

	g.It("should show no parent and one sibling for the test2 a installation", func() {
		ctx := context.Background()
		defer ctx.Done()

		inst, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperation(ctx, op, inst)
		Expect(err).ToNot(HaveOccurred())
		lCtx, err := instOp.DetermineContext(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(lCtx.Parent).To(BeNil())
		// should be 1, but this is currently a workaround until this issue https://github.com/kubernetes-sigs/controller-runtime/issues/866 is fixed
		Expect(lCtx.Siblings).To(HaveLen(2))
		//Expect(siblings[0].Name).To(Equal("b"))
	})

	g.It("should correctly determine the visible context of a installation with its parent and sibling installations", func() {
		ctx := context.Background()
		defer ctx.Done()

		inst, err := installations.CreateInternalInstallation(fakeRegistry, fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperation(ctx, op, inst)
		Expect(err).ToNot(HaveOccurred())
		lCtx, err := instOp.DetermineContext(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(lCtx.Parent).ToNot(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(3))

		Expect(lCtx.Parent.Info.Name).To(Equal("root"))
	})

})
