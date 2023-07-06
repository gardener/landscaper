// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
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
			logging.Discard(),
			testenv.Client,
			testenv.Client,
			manifestv1alpha2.Configuration{},
		)
		Expect(err).ToNot(HaveOccurred())

		ctrl = deployerlib.NewController(
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
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())
		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
		Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeTrue())
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
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		// Expect that the deploy item gets deleted
		Expect(wait.PollImmediate(5*time.Second, time.Minute, func() (done bool, err error) {
			if _, err = ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace())); err != nil {
				return false, nil
			}

			err = testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)
			return err != nil && apierrors.IsNotFound(err), nil
		})).To(Succeed())

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
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
		Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeTrue())

		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		By("update deploy item")
		di = testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/03-di-removed.yaml")
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
		Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeTrue())

		// Expect that the secret has been deleted and a configmap has been created.
		err := testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
		cm := &corev1.ConfigMap{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), cm))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())
		// Expect that the deploy item gets deleted
		Expect(wait.PollImmediate(5*time.Second, time.Minute, func() (done bool, err error) {
			if _, err = ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace())); err != nil {
				return false, nil
			}

			err = testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)
			return err != nil && apierrors.IsNotFound(err), nil
		})).To(Succeed())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), &corev1.ConfigMap{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

	It("should fail if a resource is created in non existing namespace", func() {
		ctx := context.Background()
		defer ctx.Done()

		di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/04-di-invalid.yaml")
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

		Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Progressing)).To(BeTrue())
		Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeFalse())
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
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(testutil.AddAnnotationForDeployItem(ctx, testenv, di, lsv1alpha1.DeleteWithoutUninstallAnnotation, "true")).ToNot(HaveOccurred())
		Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())
		// Expect that the deploy item gets deleted
		Expect(wait.PollImmediate(5*time.Second, time.Minute, func() (done bool, err error) {
			if _, err = ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace())); err != nil {
				return false, nil
			}

			err = testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)
			return err != nil && apierrors.IsNotFound(err), nil
		})).To(Succeed())
	})
})

func checkUpdate(pathToDI1, pathToDI2 string, state *envtest.State, ctrl reconcile.Reconciler) {
	ctx := context.Background()
	defer ctx.Done()

	By("create deploy item")
	di := testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", pathToDI1)
	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
	Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

	Expect(testutil.CreateDefaultContext(ctx, testenv.Client, nil, state.Namespace)).ToNot(HaveOccurred())

	// First reconcile will add a finalizer
	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

	Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
	Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeTrue())

	// Expect that the secret has been created
	secret := &corev1.Secret{}
	testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
	Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

	By("update deploy item")
	di = testutil.ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", pathToDI2)
	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
	Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

	testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
	Expect(utils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhases.Succeeded)).To(BeTrue())
	Expect(utils.IsDeployItemJobIDsIdentical(di)).To(BeTrue())

	// Expect that the secret has been updated
	secret = &corev1.Secret{}
	testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
	Expect(secret.Data).To(HaveKeyWithValue("config", []byte("efg")))

	testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
	Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
	Expect(testutil.UpdateJobIdForDeployItem(ctx, testenv, di, metav1.Now())).ToNot(HaveOccurred())

	// Expect that the deploy item gets deleted
	Expect(wait.PollImmediate(5*time.Second, time.Minute, func() (done bool, err error) {
		if _, err = ctrl.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace())); err != nil {
			return false, nil
		}

		err = testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)
		return err != nil && apierrors.IsNotFound(err), nil
	})).To(Succeed())

	err := testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
	Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
}
