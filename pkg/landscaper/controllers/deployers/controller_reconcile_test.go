// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers_test

import (
	"context"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/deployers"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("EnvironmentController Reconcile Test", func() {

	var (
		ctx           context.Context
		state         *envtest.State
		envController reconcile.Reconciler
		regController reconcile.Reconciler
		lsConfig      *config.LandscaperConfiguration
	)

	BeforeEach(func() {
		ctx = context.Background()
		lsConfig = &config.LandscaperConfiguration{}
		envController = deployers.NewEnvironmentController(
			testing.NullLogger{},
			testenv.Client,
			api.LandscaperScheme,
			lsConfig,
		)
		regController = deployers.NewDeployerRegistrationController(
			testing.NullLogger{},
			testenv.Client,
			api.LandscaperScheme,
			lsConfig,
		)
	})

	AfterEach(func() {
		defer ctx.Done()
		if state != nil {
			Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
			state = nil
		}
	})

	It("should automatically create a configured target", func() {
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
		lsConfig.DeployerManagement.Namespace = state.Namespace

		env := &lsv1alpha1.Environment{}
		env.GenerateName = "test"
		env.Spec.TargetSelectors = make([]lsv1alpha1.TargetSelector, 0)
		env.Spec.HostTarget.Annotations = map[string]string{
			"test": "val",
		}
		env.Spec.HostTarget.Type = "mytype"
		Expect(state.Create(ctx, testenv.Client, env)).To(Succeed())

		envReq := testutils.Request(env.Name, state.Namespace)
		testutils.ShouldReconcile(ctx, envController, envReq)

		targetList := &lsv1alpha1.TargetList{}
		Expect(testenv.Client.List(ctx, targetList, client.InNamespace(state.Namespace))).To(Succeed())
		Expect(targetList.Items).To(HaveLen(1))

		target := targetList.Items[0]
		Expect(target.Spec.Type).To(Equal(lsv1alpha1.TargetType("mytype")))
		Expect(target.Annotations).To(HaveKeyWithValue("test", "val"))
	})

	Context("Finalizer", func() {
		It("should add a finalizer to an environment", func() {
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
			lsConfig.DeployerManagement.Namespace = state.Namespace

			env := &lsv1alpha1.Environment{}
			env.GenerateName = "test"
			env.Spec.TargetSelectors = make([]lsv1alpha1.TargetSelector, 0)
			env.Spec.HostTarget.Type = "mytype"
			Expect(state.Create(ctx, testenv.Client, env)).To(Succeed())
			envKey := kutil.ObjectKeyFromObject(env)

			envReq := testutils.Request(env.Name, state.Namespace)

			testutils.ShouldReconcile(ctx, envController, envReq)

			testutils.ExpectNoError(testenv.Client.Get(ctx, envKey, env))
			Expect(controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer)).To(BeTrue())
		})

		It("should add a finalizer to an deployer registration", func() {
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
			lsConfig.DeployerManagement.Namespace = state.Namespace

			reg := &lsv1alpha1.DeployerRegistration{}
			reg.GenerateName = "test-"
			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{"test"}
			Expect(state.Create(ctx, testenv.Client, reg)).To(Succeed())
			regKey := kutil.ObjectKeyFromObject(reg)
			regReq := testutils.Request(reg.Name, state.Namespace)

			testutils.ShouldReconcile(ctx, regController, regReq)

			testutils.ExpectNoError(testenv.Client.Get(ctx, regKey, reg))
			Expect(controllerutil.ContainsFinalizer(reg, lsv1alpha1.LandscaperDMFinalizer)).To(BeTrue())
		})

		It("should remove a finalizer from the environment if all installations are deleted", func() {
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
			lsConfig.DeployerManagement.Namespace = state.Namespace

			env := &lsv1alpha1.Environment{}
			env.GenerateName = "test"
			controllerutil.AddFinalizer(env, lsv1alpha1.LandscaperDMFinalizer)
			env.Spec.TargetSelectors = make([]lsv1alpha1.TargetSelector, 0)
			env.Spec.HostTarget.Type = "mytype"
			testutils.ExpectNoError(state.Create(ctx, testenv.Client, env))
			testutils.ExpectNoError(testenv.Client.Delete(ctx, env))
			envKey := kutil.ObjectKeyFromObject(env)
			envReq := testutils.Request(env.Name, state.Namespace)

			testutils.ShouldReconcile(ctx, envController, envReq)

			err = testenv.Client.Get(ctx, envKey, env)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return
				}
				testutils.ExpectNoError(err)
			}
			Expect(controllerutil.ContainsFinalizer(env, lsv1alpha1.LandscaperDMFinalizer)).To(BeFalse())
		})

		It("should remove a finalizer from the deployer registration if all installations are deleted", func() {
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())
			lsConfig.DeployerManagement.Namespace = state.Namespace

			reg := &lsv1alpha1.DeployerRegistration{}
			reg.GenerateName = "test-"
			controllerutil.AddFinalizer(reg, lsv1alpha1.LandscaperDMFinalizer)
			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{"test"}
			testutils.ExpectNoError(state.Create(ctx, testenv.Client, reg))
			testutils.ExpectNoError(testenv.Client.Delete(ctx, reg))
			regKey := kutil.ObjectKeyFromObject(reg)
			regReq := testutils.Request(reg.Name, state.Namespace)

			testutils.ShouldReconcile(ctx, regController, regReq)

			err = testenv.Client.Get(ctx, regKey, reg)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return
				}
				testutils.ExpectNoError(err)
			}
			Expect(controllerutil.ContainsFinalizer(reg, lsv1alpha1.LandscaperDMFinalizer)).To(BeFalse())
		})
	})

})
