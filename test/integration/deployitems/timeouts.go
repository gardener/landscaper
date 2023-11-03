// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package deployitems

import (
	"context"
	"path"
	"path/filepath"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

// RegisterTests registers all tests of this package
func RegisterTests(f *framework.Framework) {
	TimeoutTestsForNewReconcile(f)
}

const (
	waitingForDeployItems   = 5 * time.Second  // how long to wait for the landscaper to create deploy items from the installation
	deployItemPickupTimeout = 30 * time.Second // the landscaper has to be configured accordingly for this test to work!
	waitingForReconcile     = 30 * time.Second // how long to wait for the landscaper or the deployer to reconcile and update the deploy item
	resyncTime              = 1 * time.Second  // after which time to check again if the condition was not fulfilled the last time
)

func maxDuration(durs ...time.Duration) time.Duration {
	var res time.Duration
	for i, d := range durs {
		if i == 0 || d > res {
			res = d
		}
	}
	return res
}

func TimeoutTestsForNewReconcile(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "deployitems", "testdata")
	)

	Describe("Deploy Item Timeouts", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %w", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should detect timeouts", func() {
			By("create test installations")
			dummy_inst := &lsv1alpha1.Installation{}
			mock_inst := &lsv1alpha1.Installation{}
			// creates deploy item without responsible deployer => pickup timeout
			utils.ExpectNoError(utils.ReadResourceFromFile(dummy_inst, path.Join(testdataDir, "00-dummy-installation.yaml")))
			// creates valid mock deploy item => no timeout
			utils.ExpectNoError(utils.ReadResourceFromFile(mock_inst, path.Join(testdataDir, "01-mock-installation.yaml")))
			// creates mock deploy item in 'Progressing' phase => first progressing timeout, then aborting timeout
			dummy_inst.SetNamespace(state.Namespace)
			mock_inst.SetNamespace(state.Namespace)
			lsv1alpha1helper.SetOperation(&dummy_inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			lsv1alpha1helper.SetOperation(&mock_inst.ObjectMeta, lsv1alpha1.ReconcileOperation)
			utils.ExpectNoError(state.Create(ctx, dummy_inst))
			utils.ExpectNoError(state.Create(ctx, mock_inst))

			By("verify that deploy items have been created")
			dummy_inst_di := &lsv1alpha1.DeployItem{}
			mock_inst_di := &lsv1alpha1.DeployItem{}
			Eventually(func() (bool, error) {
				// fetch installations
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(dummy_inst), dummy_inst)
				if err != nil || dummy_inst.Status.ExecutionReference == nil {
					return false, err
				}
				err = f.Client.Get(ctx, kutil.ObjectKeyFromObject(mock_inst), mock_inst)
				if err != nil || mock_inst.Status.ExecutionReference == nil {
					return false, err
				}
				// check for executions
				dummy_exec := &lsv1alpha1.Execution{}
				mock_exec := &lsv1alpha1.Execution{}
				err = f.Client.Get(ctx, dummy_inst.Status.ExecutionReference.NamespacedName(), dummy_exec)
				if err != nil || dummy_exec.Status.DeployItemReferences == nil || len(dummy_exec.Status.DeployItemReferences) == 0 {
					return false, err
				}
				err = f.Client.Get(ctx, mock_inst.Status.ExecutionReference.NamespacedName(), mock_exec)
				if err != nil || mock_exec.Status.DeployItemReferences == nil || len(mock_exec.Status.DeployItemReferences) == 0 {
					return false, err
				}
				// check executions for deploy item
				err = f.Client.Get(ctx, dummy_exec.Status.DeployItemReferences[0].Reference.NamespacedName(), dummy_inst_di)
				if err != nil {
					return false, err
				}
				err = f.Client.Get(ctx, mock_exec.Status.DeployItemReferences[0].Reference.NamespacedName(), mock_inst_di)
				if err != nil {
					return false, err
				}
				// return true if both deploy items could be fetched
				return true, err
			}, waitingForDeployItems, resyncTime).Should(BeTrue(), "unable to fetch deploy items")

			By("check pickup")
			Expect(deployitem.HasBeenPickedUp(dummy_inst_di)).To(BeFalse(), "dummy deploy item should not have been picked up")

			By("wait for pickup timeout to happen")
			time.Sleep(deployItemPickupTimeout)

			By("verify pickup timeout")
			// expected state:
			// - dummy_inst_di should have had a pickup timeout ('Failed' phase)
			// - mock_inst_di should be succeeded and have no reconcile timestamp annotation
			startWaitingTime := time.Now()

			Eventually(func() lsv1alpha1.DeployItemStatus {
				utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKeyFromObject(dummy_inst_di), dummy_inst_di))
				return dummy_inst_di.Status
			}, waitingForReconcile, resyncTime).Should(MatchFields(IgnoreExtras, Fields{
				"Phase": Equal(lsv1alpha1.DeployItemPhases.Failed),
				"LastError": PointTo(MatchFields(IgnoreExtras, Fields{
					"Codes":  ContainElement(lsv1alpha1.ErrorTimeout),
					"Reason": Equal(lsv1alpha1.PickupTimeoutReason),
				})),
			}), "deploy item of the dummy installation should have had a pickup timeout")

			Eventually(func() lsv1alpha1.DeployItem {
				utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKeyFromObject(mock_inst_di), mock_inst_di))
				return *mock_inst_di
			}, maxDuration(0, waitingForReconcile-time.Since(startWaitingTime)), resyncTime).Should(MatchFields(IgnoreExtras, Fields{
				"Status": MatchFields(IgnoreExtras, Fields{
					"Phase": Equal(lsv1alpha1.DeployItemPhases.Succeeded),
				}),
			}), "deploy item of the mock installation should not have had a pickup timeout")
		})
	})
}
