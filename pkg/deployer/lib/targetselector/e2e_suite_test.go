// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetselector_test

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/mock"
	"github.com/gardener/landscaper/pkg/utils"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("E2E", func() {

	var state *envtest.State

	BeforeEach(func() {
		ctx := context.Background()
		defer ctx.Done()
		var err error
		state, err = testenv.InitState(ctx)
		testutils.ExpectNoError(err)
	})

	It("should reconcile a deploy item with matching annotation selector", func() {
		ctx := context.Background()
		defer ctx.Done()

		const (
			AnnotationKey   = "somekey"
			AnnotationValue = "do"
		)

		tgt, err := utils.NewTargetBuilder("any").
			Key(state.Namespace, "target-1").
			AddAnnotation(AnnotationKey, AnnotationValue).
			Build()
		testutils.ExpectNoError(err)
		testutils.ExpectNoError(state.Create(ctx, tgt))

		mockConfig := &mockv1alpha1.ProviderConfiguration{}
		phase := lsv1alpha1.ExecutionPhaseSucceeded
		mockConfig.Phase = &phase
		di, err := mock.NewDeployItemBuilder().
			Key(state.Namespace, "test1").
			ProviderConfig(mockConfig).
			TargetFromObjectKey(kutil.ObjectKeyFromObject(tgt)).
			GenerateJobID().
			Build()
		testutils.ExpectNoError(err)
		testutils.ExpectNoError(state.Create(ctx, di, envtest.UpdateStatus(true)))

		defaultContext := &lsv1alpha1.Context{}
		defaultContext.Name = lsv1alpha1.DefaultContextName
		defaultContext.Namespace = state.Namespace
		testutils.ExpectNoError(state.Create(ctx, defaultContext))

		ctrl, err := mock.NewController(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024), mockv1alpha1.Configuration{
			TargetSelector: []lsv1alpha1.TargetSelector{
				{
					Annotations: []lsv1alpha1.Requirement{
						{
							Key:      AnnotationKey,
							Operator: selection.Equals,
							Values:   []string{AnnotationValue},
						},
					},
				},
			},
		})
		testutils.ExpectNoError(err)

		testutils.ShouldReconcile(ctx, ctrl, kutil.ReconcileRequestFromObject(di))

		resDi := &lsv1alpha1.DeployItem{}
		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), resDi))
		Expect(resDi.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(utils.IsDeployItemPhase(resDi, lsv1alpha1.DeployItemPhaseSucceeded)).To(BeTrue())
		Expect(utils.IsDeployItemJobIDsIdentical(resDi)).To(BeTrue())
	})

	It("should not reconcile a deploy item if the annotation selector does not match", func() {
		ctx := context.Background()
		defer ctx.Done()

		const (
			AnnotationKey   = "somekey"
			AnnotationValue = "do"
		)

		tgt, err := utils.NewTargetBuilder("any").
			Key(state.Namespace, "target-1").
			AddAnnotation(AnnotationKey, AnnotationValue).
			Build()
		testutils.ExpectNoError(err)
		testutils.ExpectNoError(state.Create(ctx, tgt))

		mockConfig := &mockv1alpha1.ProviderConfiguration{}
		phase := lsv1alpha1.ExecutionPhaseSucceeded
		mockConfig.Phase = &phase
		di, err := mock.NewDeployItemBuilder().
			Key(state.Namespace, "test1").
			ProviderConfig(mockConfig).
			TargetFromObjectKey(kutil.ObjectKeyFromObject(tgt)).
			Build()
		testutils.ExpectNoError(err)
		testutils.ExpectNoError(state.Create(ctx, di))

		ctrl, err := mock.NewController(logr.Discard(), testenv.Client, api.LandscaperScheme, record.NewFakeRecorder(1024), mockv1alpha1.Configuration{
			TargetSelector: []lsv1alpha1.TargetSelector{
				{
					Annotations: []lsv1alpha1.Requirement{
						{
							Key:      AnnotationKey,
							Operator: selection.Equals,
							Values:   []string{"someother"},
						},
					},
				},
			},
		})
		testutils.ExpectNoError(err)

		testutils.ShouldReconcile(ctx, ctrl, kutil.ReconcileRequestFromObject(di))

		resDi := &lsv1alpha1.DeployItem{}
		testutils.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), resDi))
		Expect(resDi.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhase("")))
	})
})
