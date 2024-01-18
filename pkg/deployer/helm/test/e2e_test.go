// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package test_test

import (
	"context"
	"time"

	"github.com/gardener/landscaper/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/apis/deployer/helm"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	helmctrl "github.com/gardener/landscaper/pkg/deployer/helm"
	deployerlib "github.com/gardener/landscaper/pkg/deployer/lib"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
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

		deployer, err := helmctrl.NewDeployer(testenv.Client, testenv.Client, testenv.Client, testenv.Client,
			logging.Discard(),
			helmv1alpha1.Configuration{},
		)
		Expect(err).ToNot(HaveOccurred())

		ctrl := deployerlib.NewController(
			testenv.Client, testenv.Client, testenv.Client, testenv.Client,
			utils.NewFinishedObjectCache(),
			api.LandscaperScheme,
			record.NewFakeRecorder(1024),
			api.LandscaperScheme,
			deployerlib.DeployerArgs{
				Type:     helmctrl.Type,
				Deployer: deployer,
			},
			5, false, "nginx-test"+testutil.GetNextCounter())

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
		di.Spec.Timeout = &lsv1alpha1.Duration{Duration: 1 * time.Second}

		di.Status.SetJobID("1")

		// Create target
		target, err := testutil.CreateOrUpdateTarget(ctx,
			testenv.Client,
			di.Spec.Target.Namespace,
			di.Spec.Target.Name,
			string(targettypes.KubernetesClusterTargetType),
			targettypes.KubernetesClusterTargetConfig{
				Kubeconfig: targettypes.ValueRef{
					StrVal: pointer.String(string(kubeconfigBytes)),
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

		di.Spec.Configuration, err = helper.ProviderConfigurationToRawExtension(providerConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(state.Create(ctx, di, envtest.UpdateStatus(true))).To(Succeed())

		// Reconcile. Provoke a timeout before the readiness check. At this point, the helm chart has been deployed,
		// and the status contains the list of managed resources.
		timeout.ActivateCheckpointTimeoutChecker(helmctrl.TimeoutCheckpointHelmBeforeReadinessCheck)
		defer timeout.ActivateStandardTimeoutChecker()
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(di), di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
		Expect(di.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))

		// Get the managed objects from status and set them in ready status
		Expect(di.Status.ProviderStatus).ToNot(BeNil())
		status := &helm.ProviderStatus{}
		helmDecoder := serializer.NewCodecFactory(helmctrl.HelmScheme).UniversalDecoder()
		_, _, err = helmDecoder.Decode(di.Status.ProviderStatus.Raw, nil, status)
		Expect(err).ToNot(HaveOccurred())
		for _, ref := range status.ManagedResources {
			obj := kutil.ObjectFromCoreObjectReference(&ref.Resource)
			Expect(testenv.Client.Get(ctx, testutil.Request(obj.GetName(), obj.GetNamespace()).NamespacedName, obj)).To(Succeed())
			Expect(testutil.SetReadyStatus(ctx, testenv.Client, obj)).To(Succeed())
		}

		// Reconcile again, now without provoking a timeout.
		// The readiness check should be successful, because we have prepared the managed objects accordingly.
		di.Status.Phase = lsv1alpha1.DeployItemPhases.Progressing
		di.Status.SetJobID(di.Status.GetJobID() + "-1")
		Expect(testenv.Client.Status().Update(ctx, di)).To(Succeed())

		timeout.ActivateIgnoreTimeoutChecker()
		testutil.ShouldReconcile(ctx, ctrl, testutil.Request(di.GetName(), di.GetNamespace()))

		Expect(testenv.Client.Get(ctx, client.ObjectKeyFromObject(di), di)).To(Succeed())
		Expect(di.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))
		deploymentList := &appsv1.DeploymentList{}
		Expect(testenv.Client.List(ctx, deploymentList, client.InNamespace(state.Namespace))).To(Succeed())
		Expect(deploymentList.Items).To(HaveLen(1))
		deployment := deploymentList.Items[0]
		Expect(deployment.Name).To(Equal("ingress-test-ingress-nginx-controller"))
	})

})
