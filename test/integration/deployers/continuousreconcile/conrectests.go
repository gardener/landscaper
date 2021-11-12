// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package continuousreconcile

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	testutils "github.com/gardener/landscaper/test/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	ContinuousReconcileTests(f)
}

// ContinuousReconcileTests test if the continuous reconciliation works for mock deploy items
func ContinuousReconcileTests(f *framework.Framework) {
	_ = Describe("ContinuousReconcileTests", func() {
		var (
			state         = f.Register()
			ctx           context.Context
			testdataDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "continuousreconcile", "testdata")
			testDuration  = 20 * time.Second // for how long the checks are executed in total
			resyncTime    = 1 * time.Second  // how often the deploy item is checked
			reconcileTime = 5 * time.Second  // after which time the deploy item should be reconciled (has to match continuous reconcile spec in deploy item)
			timeoutTime   = 30 * time.Second // how long to wait for deploy item changes
		)
		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should continuously reconcile a mock deploy item", func() {
			By("create deploy item")
			di := &lsv1alpha1.DeployItem{}
			testutils.ExpectNoError(testutils.ReadResourceFromFile(di, path.Join(testdataDir, "mock-di.yaml")))
			di.SetNamespace(state.Namespace)
			testutils.ExpectNoError(state.Create(ctx, di))

			By("wait until deploy item is Succeeded")
			Eventually(func() lsv1alpha1.ExecutionPhase {
				testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
				return di.Status.Phase
			}, timeoutTime, resyncTime).Should(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

			By(fmt.Sprintf("verify continuous reconciliation (this will take %d seconds)", testDuration/time.Second))
			startTime := time.Now()
			endTime := startTime.Add(testDuration)
			for !time.Now().After(endTime) {
				testutils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
				timeSinceLastReconcile := time.Since(di.Status.LastReconcileTime.Time)
				Expect(timeSinceLastReconcile).To(BeNumerically("<", reconcileTime+(1*time.Second))) // add one second to allow for minor imprecision and rounding
			}
		})
	})
}
