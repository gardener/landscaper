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

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/healthcheck"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Reconcile", func() {
	var (
		state *envtest.State
	)
	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.Background())
		Expect(err).ToNot(HaveOccurred())
	})

	It("should set status to ok when all replicas are available", func() {
		var err error
		ctx := logging.NewContextWithDiscard(context.Background())

		state, err = testenv.InitResources(ctx, "./testdata/test1")
		Expect(err).ToNot(HaveOccurred())

		lsDeployments := config.LsDeployments{
			LsController:         "landscaper-controller",
			LsMainController:     "landscaper-controller-main",
			WebHook:              "landscaper-webhooks",
			DeploymentsNamespace: state.Namespace,
			LsHealthCheckName:    "landscaper",
			AdditionalDeployments: &config.AdditionalDeployments{
				Deployments: []string{
					"helm",
				},
			},
		}

		healthChecker := healthcheck.NewHealthChecker(state.Client, state.Client, state.Client, state.Client, &lsDeployments)

		lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace}, lsHealthCheck)).ToNot(HaveOccurred())

		beforeReconcile := time.Now()
		time.Sleep(time.Second * 1)

		healthChecker.ExecuteHealthCheck(ctx)

		Expect(state.Client.Get(ctx, types.NamespacedName{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace}, lsHealthCheck)).ToNot(HaveOccurred())
		Expect(lsHealthCheck.Status).To(Equal(lsv1alpha1.LsHealthCheckStatusOk))
		Expect(lsHealthCheck.LastUpdateTime.Time.After(beforeReconcile)).To(BeTrue())
	})

	It("should set status to failed when not all replicas are available", func() {
		var err error
		ctx := logging.NewContextWithDiscard(context.Background())

		state, err = testenv.InitResources(ctx, "./testdata/test2")
		Expect(err).ToNot(HaveOccurred())

		lsDeployments := config.LsDeployments{
			LsController:         "landscaper-controller",
			LsMainController:     "landscaper-controller-main",
			WebHook:              "landscaper-webhooks",
			DeploymentsNamespace: state.Namespace,
			LsHealthCheckName:    "landscaper",
			AdditionalDeployments: &config.AdditionalDeployments{
				Deployments: []string{
					"helm",
				},
			},
		}

		healthChecker := healthcheck.NewHealthChecker(state.Client, state.Client, state.Client, state.Client, &lsDeployments)

		lsHealthCheck := &lsv1alpha1.LsHealthCheck{}
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace}, lsHealthCheck)).ToNot(HaveOccurred())

		healthChecker.ExecuteHealthCheck(ctx)

		// The health check only transitions to failed after two consecutive failed checks.
		Expect(state.Client.Get(ctx, types.NamespacedName{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace}, lsHealthCheck)).ToNot(HaveOccurred())

		healthChecker.ExecuteHealthCheck(ctx)

		Expect(state.Client.Get(ctx, types.NamespacedName{Name: lsDeployments.LsHealthCheckName, Namespace: lsDeployments.DeploymentsNamespace}, lsHealthCheck)).ToNot(HaveOccurred())
		Expect(lsHealthCheck.Status).To(Equal(lsv1alpha1.LsHealthCheckStatusFailed))
	})
})
