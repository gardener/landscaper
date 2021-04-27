// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helmcharts

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/gardener/landscaper/pkg/deployer/helm"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
)

// RegisterTests registers all tests of the package
func RegisterTests(f *framework.Framework) {
	DeployerTests(f)
}

// DeployerTests tests if the deployers can be deployed into a cluster through their Helm Charts and the Helm Deployer
func DeployerTests(f *framework.Framework) {
	_ = Describe("DeployerTests", func() {
		var (
			state = f.Register()
			ctx   context.Context
		)
		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should deploy the Helm-Deployer through its Helm Chart", func() {
			By("Creating and applying a Helm-Deployer DeployItem")
			var (
				chartDir   = filepath.Join(f.RootPath, "/charts/helm-deployer")
				valuesFile = filepath.Join(chartDir, "values.yaml")
			)

			const deployerName = "helm-deployer"

			di := deployDeployItemAndWaitForSuccess(ctx, f, &state.State, deployerName, chartDir, valuesFile)
			removeDeployItemAndWaitForSuccess(ctx, f, &state.State, di)
		})

		It("should deploy the Container-Deployer through its Helm Chart", func() {
			By("Creating and applying a Container Deployer DeployItem")
			var (
				chartDir   = filepath.Join(f.RootPath, "/charts/container-deployer")
				valuesFile = filepath.Join(chartDir, "values.yaml")
			)

			const deployerName = "container-deployer"

			di := deployDeployItemAndWaitForSuccess(ctx, f, &state.State, deployerName, chartDir, valuesFile)
			removeDeployItemAndWaitForSuccess(ctx, f, &state.State, di)
		})

		It("should deploy the Manifest-Deployer through its Helm Chart", func() {
			By("Creating and applying a Manifest Deployer DeployItem")
			var (
				chartDir   = filepath.Join(f.RootPath, "/charts/manifest-deployer")
				valuesFile = filepath.Join(chartDir, "values.yaml")
			)

			const deployerName = "manifest-deployer"

			di := deployDeployItemAndWaitForSuccess(ctx, f, &state.State, deployerName, chartDir, valuesFile)
			removeDeployItemAndWaitForSuccess(ctx, f, &state.State, di)
		})

		It("should deploy the Mock-Deployer through its Helm Chart", func() {
			By("Creating and applying a Mock Deployer DeployItem")
			var (
				chartDir   = filepath.Join(f.RootPath, "/charts/mock-deployer")
				valuesFile = filepath.Join(chartDir, "values.yaml")
			)

			const deployerName = "mock-deployer"

			di := deployDeployItemAndWaitForSuccess(ctx, f, &state.State, deployerName, chartDir, valuesFile)
			removeDeployItemAndWaitForSuccess(ctx, f, &state.State, di)
		})
	})
}

// deployDeployItemAndWaitForSuccess deploys a DeployItem, waits for it to succeed and for a Deployment of same name to become ready
func deployDeployItemAndWaitForSuccess(
	ctx context.Context,
	f *framework.Framework,
	state *envtest.State,
	deployerName string,
	chartDir string,
	valuesFile string) *lsv1alpha1.DeployItem {

	target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, deployerName, f.RestConfig, true)
	utils.ExpectNoError(err)
	utils.ExpectNoError(state.Create(ctx, f.Client, target))

	By("Creating the DeployItem")
	di := forgeHelmDeployItem(chartDir, valuesFile, deployerName, target, f.LsVersion)
	utils.ExpectNoError(state.Create(ctx, f.Client, di))
	By("Waiting for the DeployItem to succeed")
	utils.ExpectNoError(lsutils.WaitForDeployItemToSucceed(ctx, f.Client, di, 2*time.Minute))
	By("Waiting for the corresponding Deployment to become ready")

	// check deployment image version
	deploy := &appsv1.Deployment{}
	deployKey := kutil.ObjectKey(deployerName, state.Namespace)
	utils.ExpectNoError(f.Client.Get(ctx, deployKey, deploy))
	splitImage := strings.Split(deploy.Spec.Template.Spec.Containers[0].Image, ":")
	Expect(splitImage[len(splitImage)-1]).To(Equal(f.LsVersion))

	utils.ExpectNoError(utils.WaitForDeploymentToBeReady(ctx, f.TestLog(), f.Client, deployKey, 2*time.Minute))

	return di
}

// removeDeployItemAndWaitForSuccess removes a DeployItem and waits for it to disappear and for a Deployment of same name to get deleted
func removeDeployItemAndWaitForSuccess(
	ctx context.Context,
	f *framework.Framework,
	state *envtest.State,
	di *lsv1alpha1.DeployItem) {

	By("Removing the DeployItem")
	utils.ExpectNoError(f.Client.Delete(ctx, di))
	By("Waiting for the DeployItem to disappear")
	utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, 2*time.Minute))

	By("Waiting for the corresponding Deployment to get deleted")
	deployerDeployment := &appsv1.Deployment{}
	err := f.Client.Get(ctx, kutil.ObjectKey(di.Name, state.Namespace), deployerDeployment)
	if err != nil && !apierrors.IsNotFound(err) {
		utils.ExpectNoError(err)
	} else if err == nil {
		Expect(deployerDeployment.DeletionTimestamp.IsZero()).To(BeTrue())
	}
}

// TargetSelector for deployers deployed in these tests to make sure they do not pick up other DeployItems
const (
	targetSelectorKey      = "testing.landscaper.gardener/helmcharts"
	targetSelectorOperator = selection.Exists
)

func forgeHelmDeployItem(chartDir string, valuesFile string, name string, target *lsv1alpha1.Target, version string) *lsv1alpha1.DeployItem {
	chartBytes, closer := utils.ReadChartFrom(chartDir)
	defer closer()

	targetSelector := []lsv1alpha1.TargetSelector{
		{
			Annotations: []lsv1alpha1.Requirement{
				{
					Key:      targetSelectorKey,
					Operator: targetSelectorOperator,
				},
			},
		},
	}

	valuesBytes := utils.ReadValuesFromFile(valuesFile)
	utils.InjectTargetSelectorIntoValues(&valuesBytes, targetSelector)

	if len(version) > 0 {
		utils.InjectImageTagIntoValues(&valuesBytes, version)
	}

	chartArchive := &helmv1alpha1.Chart{
		Archive: &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartBytes),
		},
	}

	config := &helmv1alpha1.ProviderConfiguration{
		Name:      name,
		Namespace: target.Namespace,
		Values:    valuesBytes,
		Chart:     *chartArchive,
	}

	rawProviderConfig, err := helper.ProviderConfigurationToRawExtension(config)
	Expect(err).NotTo(HaveOccurred())

	di := lsv1alpha1.DeployItem{
		TypeMeta: metav1.TypeMeta{
			APIVersion: lsv1alpha1.SchemeGroupVersion.String(),
			Kind:       "DeployItem",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: target.Namespace,
		},
		Spec: lsv1alpha1.DeployItemSpec{
			Configuration: rawProviderConfig,
			Target: &lsv1alpha1.ObjectReference{
				Name:      target.Name,
				Namespace: target.Namespace,
			},
			Type: helm.Type,
		},
	}

	return &di
}
