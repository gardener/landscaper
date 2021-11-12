// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployers_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/config"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	deployers "github.com/gardener/landscaper/pkg/deployermanagement/controller"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Deployer Management Test", func() {

	var (
		ctx      context.Context
		state    *envtest.State
		dm       *deployers.DeployerManagement
		dmConfig *config.DeployerManagementConfiguration
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
		dmConfig = &config.DeployerManagementConfiguration{}
		dmConfig.Namespace = state.Namespace
		dm = deployers.NewDeployerManagement(logr.Discard(), testenv.Client, api.LandscaperScheme, *dmConfig)
	})

	AfterEach(func() {
		defer ctx.Done()
		if state != nil {
			Expect(testenv.CleanupState(ctx, state)).ToNot(HaveOccurred())
			state = nil
		}
	})

	Context("Delete", func() {
		It("should delete an installation of an environment and registration", func() {
			env := &lsv1alpha1.Environment{}
			env.GenerateName = "test-"
			env.Spec.TargetSelectors = make([]lsv1alpha1.TargetSelector, 0)
			env.Spec.HostTarget.Type = "mytype"
			Expect(state.Create(ctx, env)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			reg.GenerateName = "test-"
			controllerutil.AddFinalizer(reg, lsv1alpha1.LandscaperDMFinalizer)
			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{"test"}
			testutils.ExpectNoError(state.Create(ctx, reg))

			inst := &lsv1alpha1.Installation{}
			inst.GenerateName = "test-"
			inst.Namespace = state.Namespace
			inst.Labels = map[string]string{
				lsv1alpha1.DeployerEnvironmentLabelName:  env.Name,
				lsv1alpha1.DeployerRegistrationLabelName: reg.Name,
			}
			testutils.ExpectNoError(state.Create(ctx, inst))
			instKey := kutil.ObjectKeyFromObject(inst)

			testutils.ExpectNoError(dm.Delete(ctx, reg, env))

			err := testenv.Client.Get(ctx, instKey, inst)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsNotFound(err)).To(BeTrue(), "error should be a NotFound error")
		})
	})

	Context("Deploy", func() {
		It("should create an installation of an environment and registration", func() {
			env := &lsv1alpha1.Environment{}
			env.GenerateName = "test-"
			env.Spec.TargetSelectors = make([]lsv1alpha1.TargetSelector, 0)
			env.Spec.HostTarget.Type = "mytype"
			env.Spec.Namespace = state.Namespace
			Expect(state.Create(ctx, env)).To(Succeed())

			reg := &lsv1alpha1.DeployerRegistration{}
			reg.GenerateName = "test-"
			controllerutil.AddFinalizer(reg, lsv1alpha1.LandscaperDMFinalizer)
			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{"test"}
			testutils.ExpectNoError(state.Create(ctx, reg))

			testutils.MimicKCMServiceAccount(ctx, testenv.Client, testutils.MimicKCMServiceAccountArgs{
				Name:      deployers.FQName(reg, env),
				Namespace: state.Namespace,
				Token:     "my-service-account-token",
			})
			testutils.ExpectNoError(dm.Reconcile(ctx, reg, env))

			inst := &lsv1alpha1.Installation{}
			instKey := kutil.ObjectKey(deployers.FQName(reg, env), state.Namespace)
			testutils.ExpectNoError(testenv.Client.Get(ctx, instKey, inst))

			Expect(inst.Spec.ImportDataMappings).To(HaveKeyWithValue("releaseName",
				lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", deployers.FQName(reg, env))))))
			Expect(inst.Spec.ImportDataMappings).To(HaveKeyWithValue("releaseNamespace",
				lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", state.Namespace)))))
			Expect(inst.Spec.ImportDataMappings).To(HaveKeyWithValue("identity",
				lsv1alpha1.NewAnyJSON([]byte(fmt.Sprintf("%q", deployers.FQName(reg, env))))))

			lsTargetImport := inst.Spec.Imports.Targets[1]
			Expect(lsTargetImport.Name).To(Equal("landscaperCluster"))
			target := &lsv1alpha1.Target{}
			targetKey := kutil.ObjectKey(strings.TrimPrefix(lsTargetImport.Target, "#"), state.Namespace)
			testutils.ExpectNoError(testenv.Client.Get(ctx, targetKey, target))
			Expect(string(target.Spec.Configuration.RawMessage)).To(ContainSubstring("my-service-account-token"))
		})
	})

})
