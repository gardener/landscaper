// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifest_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/manifest"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "manifest Test Suite")
}

var (
	testenv     *envtest.Environment
	projectRoot = filepath.Join("../../../")
)

var _ = BeforeSuite(func() {
	var err error
	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())

	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

var _ = Describe("Reconcile", func() {

	var (
		ctx   context.Context
		state *envtest.State
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	It("should create a configured configmap", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha1.ProviderConfiguration{}
		manifestConfig.Manifests = []*runtime.RawExtension{
			rawCM,
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())

		m, err := manifest.New(logr.Discard(), testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cmRes := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes)).To(Succeed())
		Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
	})

	It("should create a configured configmap with a kubeconfig target referenced by secret", func() {
		kubeconfigBytes, err := kutil.GenerateKubeconfigBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		kSecret := &corev1.Secret{}
		kSecret.Name = "my-target"
		kSecret.Namespace = state.Namespace
		kSecret.Data = map[string][]byte{
			lsv1alpha1.DefaultKubeconfigKey: kubeconfigBytes,
		}
		Expect(state.Create(ctx, kSecret)).To(Succeed())

		target, err := utils.CreateKubernetesTargetFromSecret(state.Namespace, "my-target", kSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha1.ProviderConfiguration{}
		manifestConfig.Manifests = []*runtime.RawExtension{
			rawCM,
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())

		m, err := manifest.New(logr.Discard(), testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cmRes := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes)).To(Succeed())
		Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))
	})

	It("should delete a created configmap", func() {
		target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, target)).To(Succeed())

		cm := &corev1.ConfigMap{}
		cm.Name = "my-cm"
		cm.Namespace = state.Namespace
		cm.Data = map[string]string{
			"key": "val",
		}
		rawCM, err := kutil.ConvertToRawExtension(cm, scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())

		manifestConfig := &manifestv1alpha1.ProviderConfiguration{}
		manifestConfig.Manifests = []*runtime.RawExtension{
			rawCM,
		}
		item, err := manifest.NewDeployItemBuilder().
			Key(state.Namespace, "myitem").
			ProviderConfig(manifestConfig).
			Target(target.Namespace, target.Name).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(state.Create(ctx, item)).To(Succeed())

		m, err := manifest.New(logr.Discard(), testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())

		Expect(m.Reconcile(ctx)).To(Succeed())

		cmRes := &corev1.ConfigMap{}
		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes)).To(Succeed())
		Expect(cmRes.Data).To(HaveKeyWithValue("key", "val"))

		Expect(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item)).To(Succeed())
		m, err = manifest.New(logr.Discard(), testenv.Client, testenv.Client, &manifestv1alpha2.Configuration{}, item, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(m.Delete(ctx)).To(HaveOccurred())
		Expect(m.Delete(ctx)).To(Succeed())

		cmRes = &corev1.ConfigMap{}
		Expect(apierrors.IsNotFound(testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cmRes))).To(BeTrue())
	})
})
