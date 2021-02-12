// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	logtesting "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest"
	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	manifestactuator "github.com/gardener/landscaper/pkg/deployer/manifest"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Manifest Deployer", func() {

	var state *envtest.State

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should create a secret defined by a manifest deployer", func() {
		ctx := context.Background()
		defer ctx.Done()

		actuator, err := manifestactuator.NewActuator(logtesting.NullLogger{}, &manifestv1alpha1.Configuration{})
		Expect(err).ToNot(HaveOccurred())
		ok, err := inject.ClientInto(testenv.Client, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		ok, err = inject.SchemeInto(kubernetes.LandscaperScheme, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		di := ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/01-di.yaml")

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(di.Status.ProviderStatus).ToNot(BeNil(), "the provider status should be written")

		status := &manifest.ProviderStatus{}
		manifestDecoder := serializer.NewCodecFactory(manifestactuator.ManifestScheme).UniversalDecoder()
		_, _, err = manifestDecoder.Decode(di.Status.ProviderStatus.Raw, nil, status)
		testutil.ExpectNoError(err)
		Expect(status.ManagedResources).To(HaveLen(1))

		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := actuator.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

	It("should update a secret defined by a manifest deployer", func() {
		ctx := context.Background()
		defer ctx.Done()

		actuator, err := manifestactuator.NewActuator(logtesting.NullLogger{}, &manifestv1alpha1.Configuration{})
		Expect(err).ToNot(HaveOccurred())
		ok, err := inject.ClientInto(testenv.Client, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		ok, err = inject.SchemeInto(kubernetes.LandscaperScheme, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		By("create deploy item")
		di := ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/01-di.yaml")

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		By("update deploy item")
		di = ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/02-di-updated.yaml")
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		// Expect that the secret has been updated
		secret = &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("efg")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := actuator.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

	It("should cleanup resources that are removed from the list of managed resources", func() {
		ctx := context.Background()
		defer ctx.Done()

		actuator, err := manifestactuator.NewActuator(logtesting.NullLogger{}, &manifestv1alpha1.Configuration{})
		Expect(err).ToNot(HaveOccurred())
		ok, err := inject.ClientInto(testenv.Client, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		ok, err = inject.SchemeInto(kubernetes.LandscaperScheme, actuator)
		Expect(ok).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		By("create deploy item")
		di := ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/01-di.yaml")

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		// Expect that the secret has been created
		secret := &corev1.Secret{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), secret))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		By("update deploy item")
		di = ReadAndCreateOrUpdateDeployItem(ctx, testenv, state, "ingress-test-di", "./testdata/03-di-removed.yaml")
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

		// Expect that the secret has been deleted and a configmap has been created.
		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-secret", "default"), &corev1.Secret{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
		cm := &corev1.ConfigMap{}
		testutil.ExpectNoError(testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), cm))
		Expect(secret.Data).To(HaveKeyWithValue("config", []byte("abc")))

		testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		// Expect that the deploy item gets deleted
		Eventually(func() error {
			_, err := actuator.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
			return err
		}, time.Minute, 5*time.Second).Should(Succeed())

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())

		err = testenv.Client.Get(ctx, kutil.ObjectKey("my-configmap", "default"), &corev1.ConfigMap{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "secret should be deleted")
	})

})

// ReadAndCreateOrUpdateDeployItem reads a deploy item from the given file and creates or updated the deploy item
func ReadAndCreateOrUpdateDeployItem(ctx context.Context, testenv *envtest.Environment, state *envtest.State, diName, file string) *lsv1alpha1.DeployItem {
	kubeconfigBytes, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
	Expect(err).ToNot(HaveOccurred())

	di := &lsv1alpha1.DeployItem{}
	testutil.ExpectNoError(testutil.ReadResourceFromFile(di, file))
	di.Name = diName
	di.Namespace = state.Namespace
	di.Spec.Target = &lsv1alpha1.ObjectReference{
		Name:      "test-target",
		Namespace: state.Namespace,
	}

	// Create Target
	target, err := testutil.CreateOrUpdateTarget(ctx,
		testenv.Client,
		di.Spec.Target.Namespace,
		di.Spec.Target.Name,
		string(lsv1alpha1.KubernetesClusterTargetType),
		lsv1alpha1.KubernetesClusterTargetConfig{
			Kubeconfig: string(kubeconfigBytes),
		},
	)
	Expect(err).ToNot(HaveOccurred())
	Expect(state.AddResources(target)).To(Succeed())

	old := &lsv1alpha1.DeployItem{}
	if err := testenv.Client.Get(ctx, kutil.ObjectKey(di.Name, di.Namespace), old); err != nil {
		if apierrors.IsNotFound(err) {
			Expect(state.Create(ctx, testenv.Client, di, envtest.UpdateStatus(true))).To(Succeed())
			return di
		}
		testutil.ExpectNoError(err)
	}
	di.ObjectMeta = old.ObjectMeta
	testutil.ExpectNoError(testenv.Client.Patch(ctx, di, client.MergeFrom(old)))
	return di
}
