// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/gardener/landscaper/apis/config"
	testutils "github.com/gardener/landscaper/test/utils"

	"github.com/gardener/landscaper/test/utils/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/agent"
	"github.com/gardener/landscaper/pkg/api"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

var _ = Describe("Agent", func() {
	var (
		ctx      context.Context
		state    *envtest.State
		ag       *agent.Agent
		agConfig *config.AgentConfiguration
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())

		agConfig = &config.AgentConfiguration{}
		agConfig.Name = "testenv"
		agConfig.Namespace = state.Namespace
		ag = agent.New(logr.Discard(),
			testenv.Client,
			testenv.Env.Config,
			api.LandscaperScheme,
			testenv.Client,
			testenv.Env.Config,
			api.LandscaperScheme,
			*agConfig,
		)
	})

	AfterEach(func() {
		defer ctx.Done()
		// delete all environments
		env := &lsv1alpha1.Environment{}
		env.Name = agConfig.Name
		Expect(testenv.Client.Delete(ctx, env))
	})

	Context("Init", func() {
		It("should ensure all landscaper resources", func() {
			_, err := ag.EnsureLandscaperResources(ctx, testenv.Client, testenv.Client)
			Expect(err).To(Succeed())

			env := &lsv1alpha1.Environment{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey(agConfig.Name, ""), env)).To(Succeed())
		})

		It("should ensure all host resources", func() {
			testutils.MimicKCMServiceAccount(ctx, testenv.Client, testutils.MimicKCMServiceAccountArgs{
				Name:      "deployer-testenv",
				Namespace: state.Namespace,
				Token:     "test-token",
			})
			_, err := ag.EnsureHostResources(ctx, testenv.Client)
			testutils.ExpectNoError(err)

			Expect(testenv.Client.Get(ctx, kutil.ObjectKey(agent.DeployerClusterRoleName, ""), &rbacv1.ClusterRole{})).To(Succeed())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("landscaper:agent:deployer:testenv", ""), &rbacv1.ClusterRoleBinding{})).To(Succeed())

			secret := &corev1.Secret{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey(ag.TargetSecretName(), state.Namespace), secret)).To(Succeed())
			Expect(secret.Data).To(HaveKey("kubeconfig"))
			Expect(string(secret.Data["kubeconfig"])).To(ContainSubstring("test-token"), "kubeconfig should contain the newly created token")
		}, 30)
	})

})
