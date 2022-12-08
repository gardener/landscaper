// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helmchartrepo

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Helm Chart Repo Client", func() {

	var (
		ctx   context.Context
		state *envtest.State
	)

	BeforeEach(func() {
		var err error

		ctx = logging.NewContext(context.Background(), logging.Discard())
		state, err = testenv.InitState(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		defer ctx.Done()
		Expect(state.CleanupState(ctx)).To(Succeed())
	})

	It("should determine an auth header from a context", func() {
		ctx := context.Background()
		var err error

		const (
			testURL        = "test-url"
			testAuthHeader = "Basic abcdef"
		)

		helmChartRepoClient, err := NewHelmChartRepoClient(&lsv1alpha1.Context{}, nil)
		Expect(err).NotTo(HaveOccurred())

		authData := &helmv1alpha1.Auth{
			URL:        testURL,
			AuthHeader: testAuthHeader,
		}

		authHeader, err := helmChartRepoClient.getAuthHeader(ctx, authData)
		Expect(err).NotTo(HaveOccurred())
		Expect(authHeader).To(Equal(testAuthHeader))
	})

	It("should determine an auth header from a secret", func() {
		ctx := context.Background()
		var err error

		const (
			testURL        = "test-url"
			testAuthHeader = "Basic abcdef"
			testSecretName = "test-secret"
			testKey        = "test-key"
		)

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: state.Namespace,
			},
			StringData: map[string]string{
				testKey: testAuthHeader,
			},
		}
		Expect(state.Client.Create(ctx, secret)).To(Succeed())

		authData := &helmv1alpha1.Auth{
			URL: testURL,
			SecretRef: &lsv1alpha1.LocalSecretReference{
				Name: testSecretName,
				Key:  testKey,
			},
		}

		contxt := &lsv1alpha1.Context{}
		contxt.Namespace = state.Namespace
		helmChartRepoClient, err := NewHelmChartRepoClient(contxt, state.Client)
		Expect(err).NotTo(HaveOccurred())

		authHeader, err := helmChartRepoClient.getAuthHeader(ctx, authData)
		Expect(err).NotTo(HaveOccurred())
		Expect(authHeader).To(Equal(testAuthHeader))
	})

	It("should determine an auth header from a secret with default key", func() {
		ctx := context.Background()
		var err error

		const (
			testURL        = "test-url"
			testAuthHeader = "Basic abcdef"
			testSecretName = "test-secret"
		)

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: state.Namespace,
			},
			StringData: map[string]string{
				authHeaderDefaultKey: testAuthHeader,
			},
		}
		Expect(state.Client.Create(ctx, secret)).To(Succeed())

		authData := &helmv1alpha1.Auth{
			URL: testURL,
			SecretRef: &lsv1alpha1.LocalSecretReference{
				Name: testSecretName,
				// no key specified, so that authHeaderDefaultKey should be used
			},
		}

		contxt := &lsv1alpha1.Context{}
		contxt.Namespace = state.Namespace
		helmChartRepoClient, err := NewHelmChartRepoClient(contxt, state.Client)
		Expect(err).NotTo(HaveOccurred())

		authHeader, err := helmChartRepoClient.getAuthHeader(ctx, authData)
		Expect(err).NotTo(HaveOccurred())
		Expect(authHeader).To(Equal(testAuthHeader))
	})

	It("should fail to determine an auth header from a secret in another namespace", func() {
		ctx := context.Background()
		var err error

		const (
			testURL              = "test-url"
			testAuthHeader       = "Basic abcdef"
			testSecretName       = "test-secret"
			testKey              = "test-key"
			testContextNamespace = "testContextNamespace"
		)

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: state.Namespace, // differs from namespace of the context
			},
			StringData: map[string]string{
				testKey: testAuthHeader,
			},
		}
		Expect(state.Client.Create(ctx, secret)).To(Succeed())

		authData := &helmv1alpha1.Auth{
			URL: testURL,
			SecretRef: &lsv1alpha1.LocalSecretReference{
				Name: testSecretName,
				Key:  testKey,
			},
		}

		contxt := &lsv1alpha1.Context{}
		contxt.Namespace = testContextNamespace // differs from namespace of the secret
		helmChartRepoClient, err := NewHelmChartRepoClient(contxt, state.Client)
		Expect(err).NotTo(HaveOccurred())

		_, err = helmChartRepoClient.getAuthHeader(ctx, authData)
		Expect(err).To(HaveOccurred())
	})
})
