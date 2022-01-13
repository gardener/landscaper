// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/continuousreconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Manifest Deployer", func() {

	var (
		state *envtest.State
		ctrl  reconcile.Reconciler
	)

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())

		deployer, err := manifestctlr.NewDeployer(
			logr.Discard(),
			testenv.Client,
			testenv.Client,
			manifestv1alpha2.Configuration{},
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
				Type:     manifestctlr.Type,
				Deployer: deployer,
			},
		)
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should create a secret defined by a manifest deployer", func() {
		ctx := context.Background()
		defer ctx.Done()

		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/01-di.yaml")

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(di.Status.ProviderStatus).ToNot(BeNil(), "the provider status should be written")

		status := &manifestv1alpha2.ProviderStatus{}
		manifestDecoder := serializer.NewCodecFactory(manifestctlr.Scheme).UniversalDecoder()
		_, _, err := manifestDecoder.Decode(di.Status.ProviderStatus.Raw, nil, status)
		testutil.ExpectNoError(err)
		Expect(status.ManagedResources).To(HaveLen(1))

		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

	It("should update a secret defined by a manifest deployer", func() {
		checkUpdate("./testdata/01-di.yaml", "./testdata/02-di-updated.yaml", state, ctrl)
	})

	It("should patch a secret defined by a manifest deployer", func() {
		checkUpdate("./testdata/05-di.yaml", "./testdata/06-di-patched.yaml", state, ctrl)
	})

	It("should cleanup resources that are removed from the list of managed resources", func() {
		ctx := context.Background()
		defer ctx.Done()

		By("create deploy item")
		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/01-di.yaml")

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		By("update deploy item")
		di = testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/03-di-removed.yaml")
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

		// Expect that the secret has been deleted and a configmap has been created.
		err := testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
		cm := &corev1.ConfigMap{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), cm))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), &corev1.ConfigMap{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

	It("should fail if a resource is created in non existing namespace", func() {
		ctx := context.Background()
		defer ctx.Done()

		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/04-di-invalid.yaml")

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		_ = testutil.ShouldNotReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseProgressing))
		Expect(di.Status.ProviderStatus).ToNot(BeNil(), "the provider status should be written")

		status := &manifestv1alpha2.ProviderStatus{}
		manifestDecoder := serializer.NewCodecFactory(manifestctlr.Scheme).UniversalDecoder()
		_, _, err := manifestDecoder.Decode(di.Status.ProviderStatus.Raw, nil, status)
		testutil.ExpectNoError(err)
		Expect(status.ManagedResources).To(HaveLen(0))

		// Expect that the secret has not been created
		secret := &corev1.Secret{}
		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())
	})

	It("should requeue after the correct time if continuous reconciliation is configured", func() {
		ctx := context.Background()
		defer ctx.Done()

		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "conrec-test-di", "./testdata/07-di-con-rec.yaml")

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

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

func checkUpdate(pathToDI1, pathToDI2 string, state *envtest.State, ctrl reconcile.Reconciler) {
	ctx := context.Background()
	defer ctx.Done()

	By("create deploy item")
	di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", pathToDI1)

	Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

	// First reconcile will add a finalizer
	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

	Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
	// Expect that the secret has been created
	secret := &corev1.Secret{}
	testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
	Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

	By("update deploy item")
	di = testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", pathToDI2)
	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
	Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
	// Expect that the secret has been updated
	secret = &corev1.Secret{}
	testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
	Expect(secret.Data).To(HaveKeyWithValue("config", []byte("efg")))

	testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
	// Expect that the deploy item gets deleted
	Eventually(func() error {
		_, err := ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
		return err
	}, time.Minute, 5*time.Second).Should(Succeed())

	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

	err := testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
	Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
}
