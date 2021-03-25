// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitems

import (
	"context"
	"path"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/deployitem"

	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	PickupTimeoutTests(f)
}

const (
	waitingForDeployItems   = 5 * time.Second  // how long to wait for the landscaper to create deploy items from the installation
	deployItemPickupTimeout = 10 * time.Second // the landscaper has to be configured accordingly for this test to work!
	waitingForFailedState   = 10 * time.Second // how long to wait for the landscaper to set the phase to failed after the pickup timed out
	resyncTime              = 1 * time.Second  // after which time to check again if the condition was not fulfilled the last time
)

func namespacedName(meta metav1.ObjectMeta) types.NamespacedName {
	return types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      meta.Name,
	}
}

func errorListContains(list []lsv1alpha1.ErrorCode, code lsv1alpha1.ErrorCode) bool {
	for _, e := range list {
		if e == code {
			return true
		}
	}
	return false
}

func PickupTimeoutTests(f *framework.Framework) {
	var (
		dumper      = f.Register()
		testdataDir = path.Join(f.RootPath, "test", "integration", "deployitems", "testdata")
	)

	ginkgo.Describe("Deploy Item Pickup Timeout", func() {

		var (
			ctx     context.Context
			state   *envtest.State
			cleanup framework.CleanupFunc
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
			var err error
			state, cleanup, err = f.NewState(ctx)
			utils.ExpectNoError(err)
			dumper.AddNamespaces(state.Namespace)
		})

		ginkgo.AfterEach(func() {
			defer ctx.Done()
			g.Expect(cleanup(ctx)).ToNot(g.HaveOccurred())
		})

		ginkgo.It("should detect pickup timeouts", func() {
			ginkgo.By("create dummy installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "00-dummy-installation.yaml")))
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			ginkgo.By("verify that deploy items have been created")
			time.Sleep(waitingForDeployItems)
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(inst.ObjectMeta), inst))
			g.Expect(inst.Status).ToNot(g.BeNil())
			g.Expect(inst.Status.ExecutionReference).ToNot(g.BeNil())
			exec := &lsv1alpha1.Execution{}
			utils.ExpectNoError(f.Client.Get(ctx, inst.Status.ExecutionReference.NamespacedName(), exec))
			g.Expect(exec.Status).ToNot(g.BeNil())
			g.Expect(exec.Status.DeployItemReferences).ToNot(g.Or(g.BeNil(), g.BeEmpty()))
			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), di))

			ginkgo.By("check for timestamp annotation")
			// checking whether the set timestamp is up-to-date is difficult due to potential differences between the
			// system times of the machine running the landscaper and the machine running the tests
			// so just check for existence of the annotation
			g.Expect(lsv1alpha1helper.HasReconcileTimestampAnnotation(di.ObjectMeta)).To(g.BeTrue())

			ginkgo.By("check for pickup timeout")
			time.Sleep(deployItemPickupTimeout)
			success := false
			timeoutTime := time.Now().Add(waitingForFailedState)
			for { // wait for the annotation
				utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
				success = di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && di.Status.LastError != nil && errorListContains(di.Status.LastError.Codes, lsv1alpha1.ErrorTimeout) && di.Status.LastError.Reason == deployitem.PickupTimeoutReason
				if success || time.Now().After(timeoutTime) {
					// deploy item has failed due to pickup timeout
					break
				}
				time.Sleep(resyncTime)
			}
			if !success {
				// show which condition is not fulfilled
				g.Expect(di.Status.Phase).To(g.Equal(lsv1alpha1.ExecutionPhaseFailed))
				g.Expect(di.Status.LastError).NotTo(g.BeNil())
				g.Expect(errorListContains(di.Status.LastError.Codes, lsv1alpha1.ErrorTimeout)).To(g.BeTrue())
				g.Expect(di.Status.LastError.Reason).To(g.Equal(deployitem.PickupTimeoutReason))
			}
			g.Expect(success).To(g.BeTrue())
		})

		ginkgo.It("should not detect pickup timeouts for components with working deployers", func() {
			ginkgo.By("create mock installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "01-mock-installation.yaml")))
			inst.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, f.Client, inst))

			ginkgo.By("verify that deploy items have been created")
			time.Sleep(waitingForDeployItems)
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(inst.ObjectMeta), inst))
			g.Expect(inst.Status).ToNot(g.BeNil())
			g.Expect(inst.Status.ExecutionReference).ToNot(g.BeNil())
			exec := &lsv1alpha1.Execution{}
			utils.ExpectNoError(f.Client.Get(ctx, inst.Status.ExecutionReference.NamespacedName(), exec))
			g.Expect(exec.Status).ToNot(g.BeNil())
			g.Expect(exec.Status.DeployItemReferences).ToNot(g.Or(g.BeNil(), g.BeEmpty()))
			di := &lsv1alpha1.DeployItem{}
			utils.ExpectNoError(f.Client.Get(ctx, exec.Status.DeployItemReferences[0].Reference.NamespacedName(), di))

			ginkgo.By("verify that deploy item is not timed out")
			time.Sleep(deployItemPickupTimeout + waitingForFailedState) // wait for a potential timeout to happen
			utils.ExpectNoError(f.Client.Get(ctx, namespacedName(di.ObjectMeta), di))
			// check that deploy item does not have a pickup timeout
			g.Expect(di.Status.Phase == lsv1alpha1.ExecutionPhaseFailed && di.Status.LastError != nil && errorListContains(di.Status.LastError.Codes, lsv1alpha1.ErrorTimeout) && di.Status.LastError.Reason == deployitem.PickupTimeoutReason).To(g.BeFalse())
		})

	})

}
