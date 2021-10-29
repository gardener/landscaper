// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Container Deployer", func() {

	var (
		state *envtest.State
		ctrl  reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())

		deployer, err := containerctlr.NewDeployer(
			logr.Discard(),
			testenv.Client,
			testenv.Client,
			testenv.Client,
			containerv1alpha1.Configuration{},
		)
		Expect(err).ToNot(HaveOccurred())

		ctrl = deployerlib.NewController(
			logr.Discard(),
			testenv.Client,
			api.LandscaperScheme,
			record.NewFakeRecorder(1024),
			testenv.Client,
			api.LandscaperScheme,
			deployerlib.DeployerArgs{
				Type:     containerctlr.Type,
				Deployer: deployer,
			},
		)

	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should requeue after the correct time if continuous reconciliation is configured", func() {
		ctx := context.Background()
		defer ctx.Done()

		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "conrec-test-di", "./testdata/00-sleep-container.yaml")
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
		di.Status.Phase = lsv1alpha1.ExecutionPhaseSucceeded
		di.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
		di.Status.ObservedGeneration = 1
		testutil.ExpectNoError(testenv.Client.Status().Update(ctx, di))

		// reconcile once to generate status
		recRes, err := ctrl.Reconcile(ctx, kutil.ReconcileRequestFromObject(di))
		testutil.ExpectNoError(err)

		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
		lastReconciled := di.Status.LastReconcileTime
		testDuration := time.Duration(1 * time.Hour)
		expectedNextReconcileIn := time.Until(lastReconciled.Add(testDuration))
		recRes, err = ctrl.Reconcile(ctx, kutil.ReconcileRequestFromObject(di))
		testutil.ExpectNoError(err)
		timeDiff := expectedNextReconcileIn - recRes.RequeueAfter
		Expect(timeDiff).To(BeNumerically("~", time.Duration(0), 1*time.Second)) // allow for slight imprecision

		// check again when closer to the next reconciliation time
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di))
		shortTestDuration := time.Duration(10 * time.Minute)
		lastReconciled.Time = time.Now().Add((-1) * testDuration).Add(shortTestDuration)
		di.Status.LastReconcileTime = lastReconciled
		testutil.ExpectNoError(testenv.Client.Status().Update(ctx, di))
		recRes, err = ctrl.Reconcile(ctx, kutil.ReconcileRequestFromObject(di))
		testutil.ExpectNoError(err)
		lstr := di.Status.LastReconcileTime.Time.String()
		nxtr := recRes.RequeueAfter.String()
		By("last: " + lstr + " - next: " + nxtr)
		timeDiff = shortTestDuration - recRes.RequeueAfter
		Expect(timeDiff).To(BeNumerically("~", time.Duration(0), 1*time.Second)) // allow for slight imprecision

		// verify that continuous reconciliation can be disabled by annotation
		if di.Annotations == nil {
			di.Annotations = make(map[string]string)
		}
		di.Annotations[continuousreconcile.ContinuousReconcileActiveAnnotation] = "false"
		testutil.ExpectNoError(testenv.Client.Update(ctx, di))
		recRes, err = ctrl.Reconcile(ctx, kutil.ReconcileRequestFromObject(di))
		testutil.ExpectNoError(err)
		Expect(recRes.RequeueAfter).To(BeNumerically("==", time.Duration(0)))
	})

})
