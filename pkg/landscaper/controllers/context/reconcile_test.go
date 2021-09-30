// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package context_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"

	"github.com/gardener/landscaper/pkg/api"
	contextctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/context"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("Reconcile", func() {

	var (
		ctrl    reconcile.Reconciler
		state   *envtest.State
		repoCtx *cdv2.UnstructuredTypedObject
	)
	BeforeEach(func() {
		uObj, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com", ""))
		repoCtx = &uObj

		var err error
		ctrl, err = contextctrl.NewDefaulterController(
			logr.Discard(),
			testenv.Client,
			api.Scheme,
			record.NewFakeRecorder(1024),
			config.ContextControllerConfig{
				Default: config.ContextControllerDefaultConfig{
					RepositoryContext: repoCtx,
				},
			})
		Expect(err).ToNot(HaveOccurred())
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	It("should create a new default context object in a new namespace", func() {
		ctx := context.Background()
		ns := &corev1.Namespace{}
		ns.Name = state.Namespace

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(ns))

		items := &lsv1alpha1.ContextList{}
		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
		Expect(items.Items).To(HaveLen(1))
		lsCtx := &items.Items[0]
		Expect(lsCtx.Name).To(Equal(lsv1alpha1.DefaultContextName))
		Expect(lsCtx.RepositoryContext.Raw).To(MatchJSON(repoCtx.Raw))
	})

	It("should update the default config object if changed", func() {
		ctx := context.Background()
		uObj, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.de", ""))

		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = state.Namespace
		lsCtx.RepositoryContext = &uObj
		Expect(state.Create(ctx, lsCtx)).To(Succeed())

		ns := &corev1.Namespace{}
		ns.Name = state.Namespace
		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(ns))

		items := &lsv1alpha1.ContextList{}
		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
		Expect(items.Items).To(HaveLen(1))
		lsCtx = &items.Items[0]
		Expect(lsCtx.Name).To(Equal(lsv1alpha1.DefaultContextName))
		Expect(lsCtx.RepositoryContext.Raw).To(MatchJSON(repoCtx.Raw))
	})

	It("should not create a new default context object in a excluded namespace", func() {
		ctx := context.Background()

		ctrl, err := contextctrl.NewDefaulterController(
			logr.Discard(),
			testenv.Client,
			api.Scheme,
			record.NewFakeRecorder(1024),
			config.ContextControllerConfig{
				Default: config.ContextControllerDefaultConfig{
					RepositoryContext:  repoCtx,
					ExcludedNamespaces: []string{state.Namespace},
				},
			})
		Expect(err).ToNot(HaveOccurred())

		ns := &corev1.Namespace{}
		ns.Name = state.Namespace

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(ns))

		items := &lsv1alpha1.ContextList{}
		testutils.ExpectNoError(testenv.Client.List(ctx, items, client.InNamespace(state.Namespace)))
		Expect(items.Items).To(HaveLen(0))
	})

})
