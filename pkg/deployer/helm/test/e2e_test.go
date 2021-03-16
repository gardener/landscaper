// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"

	logtesting "github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	helmactuator "github.com/gardener/landscaper/pkg/deployer/helm"
	"github.com/gardener/landscaper/pkg/kubernetes"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Helm Deployer", func() {

	var state *envtest.State

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should deploy a ingress-nginx chart from a oci artifact into the cluster", func() {
		ctx := context.Background()
		defer ctx.Done()

		actuator, err := helmactuator.NewController(
			logtesting.NullLogger{},
			testenv.Client,
			kubernetes.LandscaperScheme,
			&helmv1alpha1.Configuration{},
		)
		Expect(err).ToNot(HaveOccurred())

		kubeconfigBytes, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())

		di := &lsv1alpha1.DeployItem{}
		di.Name = "ingress-test-di"
		di.Namespace = state.Namespace
		di.Spec.Target = &lsv1alpha1.ObjectReference{
			Name:      "test-target",
			Namespace: state.Namespace,
		}
		di.Spec.Type = helmactuator.Type

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

		// create helm provider config
		providerConfig := &helmv1alpha1.ProviderConfiguration{}
		providerConfig.Chart.Ref = "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v0.1.0"
		providerConfig.Name = "ingress-test"
		providerConfig.Namespace = state.Namespace

		di.Spec.Configuration, err = helper.ProviderConfigurationToRawExtension(providerConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(state.Create(ctx, testenv.Client, di, envtest.UpdateStatus(true))).To(Succeed())

		// First reconcile will add a finalizer
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))
		testutil.ShouldReconcile(ctx, actuator, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))

		deploymentList := &appsv1.DeploymentList{}
		Expect(testenv.Client.List(ctx, deploymentList, client.InNamespace(state.Namespace))).To(Succeed())
		Expect(deploymentList.Items).To(HaveLen(1))

		deployment := deploymentList.Items[0]
		Expect(deployment.Name).To(Equal("ingress-test-ingress-nginx-controller"))

		//testutil.ExpectNoError(testenv.Client.Delete(ctx, di))
		//// Expect that the deploy item gets deleted
		//Eventually(func() error{
		//	_, err := actuator.Reconcile(ctx, testutil.Request(di.GetName(), di.GetNamespace()))
		//	return err
		//}, time.Minute, 5 *time.Second).Should(Succeed())
		//
		//Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(HaveOccurred())
	})

})
