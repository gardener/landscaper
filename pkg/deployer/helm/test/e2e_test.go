// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	lsutils "github.com/gardener/landscaper/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	helmctrl "github.com/gardener/landscaper/pkg/deployer/helm"
	testutil "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Helm Deployer", func() {

	var state *envtest.State

	BeforeEach(func() {
		var err error
		state, err = testenv.InitState(context.TODO())
		Expect(err).ToNot(HaveOccurred())
		Expect(testutil.CreateExampleDefaultContext(context.TODO(), testenv.Client, state.Namespace)).To(Succeed())
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should deploy an ingress-nginx chart from an oci artifact into the cluster", func() {
		ctx := context.Background()
		defer ctx.Done()

		deployer, err := helmctrl.NewDeployer(
			logging.Discard(),
			testenv.Client,
			testenv.Client,
			helmv1alpha1.Configuration{},
		)
		Expect(err).ToNot(HaveOccurred())

		ctrl := deployerlib.NewController(
			testenv.Client,
			api.LandscaperScheme,
			record.NewFakeRecorder(1024),
			testenv.Client,
			api.LandscaperScheme,
			deployerlib.DeployerArgs{
				Type:     helmctrl.Type,
				Deployer: deployer,
			},
		)

		kubeconfigBytes, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())

		di := &lsv1alpha1.DeployItem{}
		di.Name = "ingress-test-di"
		di.Namespace = state.Namespace
		di.Spec.Target = &lsv1alpha1.ObjectReference{
			Name:      "test-target",
			Namespace: state.Namespace,
		}
		di.Spec.Type = helmctrl.Type

		di.Status.JobID = "1"

		// Create target
		target, err := testutil.CreateOrUpdateTarget(ctx,
			testenv.Client,
			di.Spec.Target.Namespace,
			di.Spec.Target.Name,
			string(lsv1alpha1.KubernetesClusterTargetType),
			lsv1alpha1.KubernetesClusterTargetConfig{
				Kubeconfig: lsv1alpha1.ValueRef{
					StrVal: pointer.StringPtr(string(kubeconfigBytes)),
				},
			},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(state.AddResources(target)).To(Succeed())

		// create helm provider config
		providerConfig := &helmv1alpha1.ProviderConfiguration{}
		providerConfig.HelmDeployment = pointer.Bool(false)
		providerConfig.Chart.Ref = "eu.gcr.io/gardener-project/landscaper/tutorials/charts/ingress-nginx:v3.29.0"
		providerConfig.Name = "ingress-test"
		providerConfig.Namespace = state.Namespace
		providerConfig.ReadinessChecks = health.ReadinessCheckConfiguration{
			Timeout: &lsv1alpha1.Duration{Duration: 1 * time.Second},
		}

		di.Spec.Configuration, err = helper.ProviderConfigurationToRawExtension(providerConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(state.Create(ctx, di, envtest.UpdateStatus(true))).To(Succeed())

		// At this stage, resources are not yet ready
		err = testutil.ShouldNotReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(err).To(HaveOccurred())

		// Get the managed objects from Status and set them in Ready status
		status := &helm.ProviderStatus{}
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())
		Expect(di.Status.ProviderStatus).ToNot(BeNil())

		helmDecoder := serializer.NewCodecFactory(helmctrl.HelmScheme).UniversalDecoder()
		_, _, err = helmDecoder.Decode(di.Status.ProviderStatus.Raw, nil, status)
		Expect(err).ToNot(HaveOccurred())

		for _, ref := range status.ManagedResources {
			obj := kutil.ObjectFromCoreObjectReference(&ref.Resource)
			Expect(testenv.Client.Get(ctx, testutil.Request(obj.GetName(), obj.GetNamespace()).NamespacedName, obj)).To(Succeed())
			Expect(testutil.SetReadyStatus(ctx, testenv.Client, obj)).To(Succeed())
		}
		di.Status.Phase = lsv1alpha1.ExecutionPhaseProgressing
		di.Status.JobID = di.Status.JobID + "-1"
		Expect(testenv.Client.Status().Update(ctx, di)).To(Succeed())

		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))
		Expect(testenv.Client.Get(ctx, testutil.Request(di.GetName(), di.GetNamespace()).NamespacedName, di)).To(Succeed())

		Expect(di.Status.Phase).To(Equal(lsv1alpha1.ExecutionPhaseSucceeded))
		Expect(lsutils.IsDeployItemPhase(di, lsv1alpha1.DeployItemPhaseSucceeded)).To(BeTrue())

		deploymentList := &appsv1.DeploymentList{}
		Expect(testenv.Client.List(ctx, deploymentList, client.InNamespace(state.Namespace))).To(Succeed())
		Expect(deploymentList.Items).To(HaveLen(1))

		deployment := deploymentList.Items[0]
		Expect(deployment.Name).To(Equal("ingress-test-ingress-nginx-controller"))
	})

})
