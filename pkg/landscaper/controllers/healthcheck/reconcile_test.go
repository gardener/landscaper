// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthcheck_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/healthcheck"
	testutils "github.com/gardener/landscaper/test/utils"

	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Reconcile", func() {
	var (
		ctrl  reconcile.Reconciler
		state *envtest.State
	)
	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.Background())
		Expect(err).ToNot(HaveOccurred())
	})

	It("should set status to ok when all replicas are available", func() {
		var err error
		ctx := context.Background()

		state, err = testenv.InitResources(ctx, "./testdata/test1")
		Expect(err).ToNot(HaveOccurred())

		agentConfig := config.AgentConfiguration{
			Name:                "landscaper",
			Namespace:           state.Namespace,
			LandscaperNamespace: "ls-system",
		}

		lsDeployments := config.LsDeployments{
			LsController: "landscaper-controller",
			WebHook:      "landscaper-webhooks",
		}

		ctrl = healthcheck.NewLsHealthCheckController(logging.Discard(), &agentConfig, &lsDeployments, state.Client, api.Scheme, []string{"helm"}, 1*time.Second)

		lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: agentConfig.Name, Namespace: state.Namespace}, lsHealthCheck)).ToNot(HaveOccurred())

		beforeReconcile := time.Now()
		time.Sleep(time.Second * 1)

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(lsHealthCheck))

		Expect(state.Client.Get(ctx, types.NamespacedName{Name: agentConfig.Name, Namespace: state.Namespace}, lsHealthCheck)).ToNot(HaveOccurred())
		Expect(lsHealthCheck.Status).To(Equal(lsv1alpha1.LsHealthCheckStatusOk))
		Expect(lsHealthCheck.LastUpdateTime.Time.After(beforeReconcile)).To(BeTrue())
	})

	It("should set status to failed when not all replicas are available", func() {
		var err error
		ctx := context.Background()

		state, err = testenv.InitResources(ctx, "./testdata/test2")
		Expect(err).ToNot(HaveOccurred())

		agentConfig := config.AgentConfiguration{
			Name:                "landscaper",
			Namespace:           state.Namespace,
			LandscaperNamespace: "ls-system",
		}

		lsDeployments := config.LsDeployments{
			LsController: "landscaper-controller",
			WebHook:      "landscaper-webhooks",
		}

		ctrl = healthcheck.NewLsHealthCheckController(logging.Discard(), &agentConfig, &lsDeployments, state.Client, api.Scheme, []string{"helm"}, 1*time.Second)

		lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: agentConfig.Name, Namespace: state.Namespace}, lsHealthCheck)).ToNot(HaveOccurred())

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(lsHealthCheck))

		// The health check only transitions to failed after two consecutive failed checks.
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: agentConfig.Name, Namespace: state.Namespace}, lsHealthCheck)).ToNot(HaveOccurred())

		time.Sleep(2 * time.Second)

		testutils.ShouldReconcile(ctx, ctrl, testutils.RequestFromObject(lsHealthCheck))
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: agentConfig.Name, Namespace: state.Namespace}, lsHealthCheck)).ToNot(HaveOccurred())
		Expect(lsHealthCheck.Status).To(Equal(lsv1alpha1.LsHealthCheckStatusFailed))
	})
})
