// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helmdeployer

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func HelmDeployerTests(f *framework.Framework) {
	_ = Describe("HelmDeployerTests", func() {
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
		Context("should deploy a helm chart with a single config map", func() {
			testFunc := func(realHelmDeployer bool) {
				By("Creating and applying a Helm-deployer DeployItem")
				var (
					chartDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01", "chart")
					valuesFile = filepath.Join(chartDir, "values.yaml")
				)

				const deployName = "configmap-deployment"
				di, err := deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, nil, nil, realHelmDeployer)
				Expect(err).ShouldNot(HaveOccurred())

				cm := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", state.Namespace), cm)).To(Succeed())
				Expect(cm.Data["key"]).To(Equal("value"))

				removeDeployItemAndWaitForSuccess(ctx, f, state.State, di)
			}

			It("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})
		Context("should update a helm chart with a single config map", func() {
			testFunc := func(realHelmDeployer bool) {
				By("Creating and applying a Helm-deployer DeployItem")
				var (
					chartDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01", "chart")
					valuesFile = filepath.Join(chartDir, "values.yaml")
				)

				const deployName = "configmap-deployment"
				deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, nil, nil, realHelmDeployer)
				cm := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", state.Namespace), cm)).To(Succeed())
				Expect(cm.Data["key"]).To(Equal("value"))

				By("Updating a Helm-deployer DeployItem")
				chartDir = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01_updated", "chart")
				di := updateDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, realHelmDeployer)
				cm_updated := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", state.Namespace), cm_updated)).To(Succeed())
				Expect(cm_updated.Data["key"]).To(Equal("value_updated"))

				removeDeployItemAndWaitForSuccess(ctx, f, state.State, di)
			}
			It("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})
		Context("should export a value from the config map", func() {
			testFunc := func(realHelmDeployer bool) {
				By("Creating and applying a Helm-deployer DeployItem")
				var (
					chartDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "02_export", "chart")
					valuesFile = filepath.Join(chartDir, "values.yaml")
				)

				const deployName = "configmap-deployment"
				exports := managedresource.Exports{
					Exports: []managedresource.Export{
						{
							Key:      "exportKey",
							JSONPath: "data.key",
							FromResource: &lsv1alpha1.TypedObjectReference{
								APIVersion: "v1",
								Kind:       "ConfigMap",
								ObjectReference: lsv1alpha1.ObjectReference{
									Name:      "mychart-configmap",
									Namespace: state.Namespace,
								},
							},
						},
					},
				}
				di, err := deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, &exports, nil, realHelmDeployer)
				Expect(err).ShouldNot(HaveOccurred())
				cm := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", state.Namespace), cm)).To(Succeed())
				Expect(cm.Data["key"]).To(Equal("value"))
				Expect(di.Status.ExportReference.Name).To(Equal("configmap-deployment-export"))

				export_secret := &corev1.Secret{}
				Expect(state.Client.Get(ctx, di.Status.ExportReference.NamespacedName(), export_secret)).To(Succeed())
				Expect(export_secret.Data[lsv1alpha1.DataObjectSecretDataKey]).ToNot(BeNil())
				exportRaw := export_secret.Data[lsv1alpha1.DataObjectSecretDataKey]

				var export map[string]interface{}
				Expect(json.Unmarshal(exportRaw, &export)).ToNot(HaveOccurred())
				Expect(export).To(HaveKeyWithValue("exportKey", "value"))

				removeDeployItemAndWaitForSuccess(ctx, f, state.State, di)
			}
			It("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})

		Context("should succeed a readinesCheck on a deployed helm chart with a single config map", func() {
			testFunc := func(realHelmDeployer bool) {
				By("Creating and applying a Helm-deployer DeployItem")
				var (
					chartDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01", "chart")
					valuesFile = filepath.Join(chartDir, "values.yaml")
				)
				const deployName = "helm-deployer"

				jsonRawMessage, err := json.Marshal(map[string]string{"value": "value"})
				Expect(err).ShouldNot(HaveOccurred())
				readinesCheck := getReadinessCheck(state.Namespace, jsonRawMessage)

				di, err := deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, nil, &readinesCheck, realHelmDeployer)
				Expect(err).ShouldNot(HaveOccurred())
				cm := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", state.Namespace), cm)).To(Succeed())
				Expect(cm.Data["key"]).To(Equal("value"))

				removeDeployItemAndWaitForSuccess(ctx, f, state.State, di)
			}
			It("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})

		Context("should fail a readinesCheck for a wrong value on a deployed helm chart with a single config map", func() {
			testFunc := func(realHelmDeployer bool) {
				By("Creating and applying a Helm-deployer DeployItem")
				var (
					chartDir   = filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01", "chart")
					valuesFile = filepath.Join(chartDir, "values.yaml")
				)
				const deployName = "helm-deployer"

				jsonRawMessage, err := json.Marshal(map[string]string{"value": "value_WRONG"})
				Expect(err).ShouldNot(HaveOccurred())
				readinesCheck := getReadinessCheck(state.Namespace, jsonRawMessage)

				di, err := deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, nil, &readinesCheck, realHelmDeployer)
				Expect(err).Should(HaveOccurred())

				removeDeployItemAndWaitForSuccess(ctx, f, state.State, di)
			}
			XIt("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})

	})
}

// deployDeployItemAndWaitForSuccess deploys a DeployItem, returns errors
func deployDeployItemAndWaitForSuccess(
	ctx context.Context,
	f *framework.Framework,
	state *envtest.State,
	chartName string,
	chartDir string,
	valuesFile string, exports *managedresource.Exports, readinesCheck *readinesschecks.ReadinessCheckConfiguration, useHelmDeployment bool) (*lsv1alpha1.DeployItem, error) {

	target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, chartName, f.RestConfig, true)
	if err != nil {
		return nil, err
	}
	err = state.Create(ctx, target)
	if err != nil {
		return nil, err
	}

	By("Creating the DeployItem")
	chartYaml := utils.ReadValuesFromFile(filepath.Join(chartDir, "Chart.yaml"))
	fmt.Fprintf(GinkgoWriter, "Chart: %s", chartYaml)

	di := createHelmDeployItem(chartDir, valuesFile, chartName, target, exports, readinesCheck, useHelmDeployment)
	err = state.Create(ctx, di)
	if err != nil {
		return di, err
	}
	// Set a new jobID to trigger a reconcile of the deploy item
	Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
	Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
	By("Waiting for the DeployItem to succeed")
	err = lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 1*time.Minute)

	return di, err
}

// updateDeployItemAndWaitForSuccess updates a DeployItem, waits for it to succeed
func updateDeployItemAndWaitForSuccess(
	ctx context.Context,
	f *framework.Framework,
	state *envtest.State,
	chartName string,
	chartDir string,
	valuesFile string, useHelmDeployment bool) *lsv1alpha1.DeployItem {

	target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, chartName, f.RestConfig, true)
	utils.ExpectNoError(err)

	By("Creating the updated DeployItem")
	chartYaml := utils.ReadValuesFromFile(filepath.Join(chartDir, "Chart.yaml"))
	fmt.Fprintf(GinkgoWriter, "Chart: %s", chartYaml)

	By("Merge existing with updated spec DeployItem")
	di := &lsv1alpha1.DeployItem{}
	state.Client.Get(ctx, kutil.ObjectKey(chartName, state.Namespace), di)
	new_di := createHelmDeployItem(chartDir, valuesFile, chartName, target, nil, nil, useHelmDeployment)
	di.Spec = new_di.Spec

	utils.ExpectNoError(state.Update(ctx, di))
	// Set a new jobID to trigger a reconcile of the deploy item
	Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
	Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
	By("Waiting for the DeployItem Update to succeed")
	utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhaseSucceeded, 20*time.Second))
	return di
}

// removeDeployItemAndWaitForSuccess removes a DeployItem and waits for it to disappear
func removeDeployItemAndWaitForSuccess(
	ctx context.Context,
	f *framework.Framework,
	state *envtest.State,
	di *lsv1alpha1.DeployItem) {

	By("Removing the DeployItem")
	utils.ExpectNoError(f.Client.Delete(ctx, di))
	// Set a new jobID to trigger a reconcile of the deploy item
	Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
	Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())

	By("Waiting for the DeployItem to disappear")
	utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, 2*time.Minute))
}

func getReadinessCheck(namespace string, jsonRawMessage []byte) readinesschecks.ReadinessCheckConfiguration {
	return readinesschecks.ReadinessCheckConfiguration{
		Timeout:        &lsv1alpha1.Duration{Duration: time.Second * 5},
		DisableDefault: true,
		CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
			{
				Name:    "valueIsCorrect",
				Timeout: &lsv1alpha1.Duration{Duration: time.Second * 5},
				Resource: []lsv1alpha1.TypedObjectReference{
					{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						ObjectReference: lsv1alpha1.ObjectReference{
							Name:      "mychart-configmap",
							Namespace: namespace,
						},
					},
				},
				Requirements: []readinesschecks.RequirementSpec{
					{
						JsonPath: ".data.key",
						Operator: selection.Equals,
						Value: []runtime.RawExtension{
							{
								Raw: jsonRawMessage,
							},
						},
					},
				},
			},
		},
	}
}

func createHelmDeployItem(chartDir string, valuesFile string, name string, target *lsv1alpha1.Target, exports *managedresource.Exports, readinesCheck *readinesschecks.ReadinessCheckConfiguration, useHelmDeployment bool) *lsv1alpha1.DeployItem {
	chartBytes, closer := utils.ReadChartFrom(chartDir)
	defer closer()

	valuesBytes := utils.ReadValuesFromFile(valuesFile)

	chartArchive := &helmv1alpha1.Chart{
		Archive: &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartBytes),
		},
	}

	config := &helmv1alpha1.ProviderConfiguration{
		Name:           name,
		Namespace:      target.Namespace,
		HelmDeployment: pointer.BoolPtr(useHelmDeployment),
		Values:         valuesBytes,
		Chart:          *chartArchive,
		Exports:        exports,
	}
	if readinesCheck != nil {
		config.ReadinessChecks = *readinesCheck
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
