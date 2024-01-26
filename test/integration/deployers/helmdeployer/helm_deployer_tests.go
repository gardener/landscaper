// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helmdeployer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/utils/ptr"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo/v2"
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

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %w", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
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
				_, err := deployDeployItemAndWaitForSuccess(ctx, f, state.State, deployName, chartDir, valuesFile, nil, nil, realHelmDeployer)
				Expect(err).ShouldNot(HaveOccurred())
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
			It("with real helm deployer", func() {
				testFunc(true)
			})
			It("with helm templating and manifest apply", func() {
				testFunc(false)
			})
		})

		Context("private registry", Ordered, func() {
			// The following test should run ordered, as check and prepare several prerequisites (registry is actually
			// private, the image required for the test is actually uploaded, ...) before running the actual test
			// (helm deployer deploying an image from a private registry).
			var imageData []byte
			var httpclient *http.Client

			var ns *corev1.Namespace
			var target *lsv1alpha1.Target
			var secret *corev1.Secret
			var lsctx *lsv1alpha1.Context
			var di *lsv1alpha1.DeployItem

			BeforeAll(func() {
				var err error
				By("configure http client and data")
				imageFile, err := os.Open(filepath.Join(f.RootPath, "test", "integration", "deployers", "helmdeployer", "testdata", "01", "chart.tgz"))
				Expect(err).To(BeNil())

				imageData, err = io.ReadAll(imageFile)
				Expect(err).To(BeNil())

				certFile, err := os.Open(f.RegistryCAPath)
				Expect(err).To(BeNil())

				certData, err := io.ReadAll(certFile)
				Expect(err).To(BeNil())

				certPool := x509.NewCertPool()
				certPool.AppendCertsFromPEM(certData)

				transport := &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: certPool,
					},
				}
				httpclient = &http.Client{Transport: transport}

				By("configure kubernetes objects")
				dockerconfigFile, err := os.Open(f.RegistryConfigPath)
				Expect(err).To(BeNil())
				Expect(dockerconfigFile).ToNot(BeNil())

				dockerconfigData, err := io.ReadAll(dockerconfigFile)
				Expect(err).To(BeNil())
				Expect(dockerconfigData).ToNot(BeNil())

				secret = &corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "private-registry-key",
						Namespace: "private-registry",
					},
					Immutable: nil,
					Type:      "kubernetes.io/dockerconfigjson",
					Data:      map[string][]byte{".dockerconfigjson": dockerconfigData},
				}

				ns = &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "private-registry",
					},
					Spec:   corev1.NamespaceSpec{},
					Status: corev1.NamespaceStatus{},
				}

				target, err = utils.BuildInternalKubernetesTarget(ctx, f.Client, "private-registry", "target", f.RestConfig, true)
				Expect(err).To(BeNil())
				Expect(target).ToNot(BeNil())

				repoCtx := &cdv2.UnstructuredTypedObject{}
				Expect(repoCtx.UnmarshalJSON([]byte(fmt.Sprintf(`{"type": "OCIRegistry", "baseUrl": "%s"}`, f.RegistryBasePath)))).To(BeNil())
				lsctx = &lsv1alpha1.Context{
					TypeMeta: metav1.TypeMeta{
						APIVersion: lsv1alpha1.SchemeGroupVersion.String(),
						Kind:       "Context",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "private-registry-context",
						Namespace: "private-registry",
					},
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						RepositoryContext:   repoCtx,
						UseOCM:              true,
						RegistryPullSecrets: []corev1.LocalObjectReference{{Name: "private-registry-key"}},
					},
				}
			})

			It("registry is private (denies access without credentials)", func() {
				By("configure helm client without credentials")
				helmClient, err := registry.NewClient(
					registry.ClientOptHTTPClient(httpclient))
				Expect(err).To(BeNil())

				By("push with helm client without credentials (expect access denied)")
				pushResult, err := helmClient.Push(imageData, f.RegistryBasePath+"/test-chart:v0.1.0")
				Expect(err).ToNot(BeNil())
				Expect(pushResult).To(BeNil())
			})
			It("registry contains image (upload succeeds)", func() {
				By("configure helm client with credentials")
				helmClient, err := registry.NewClient(
					registry.ClientOptCredentialsFile(f.RegistryConfigPath),
					registry.ClientOptHTTPClient(httpclient))
				Expect(err).To(BeNil())

				By("push with helm client with credentials (expect push to succeed)")
				pushResult, err := helmClient.Push(imageData, f.RegistryBasePath+"/test-chart:v0.1.0")
				Expect(err).To(BeNil())
				Expect(pushResult).ToNot(BeNil())
			})

			testFunc := func(realHelmDeployer bool) {

				By("configure deploy item")
				config := &helmv1alpha1.ProviderConfiguration{
					Name:           "privreg-deploy-item",
					Namespace:      "private-registry",
					HelmDeployment: ptr.To[bool](realHelmDeployer),
					Values:         nil,
					Chart:          helmv1alpha1.Chart{Ref: f.RegistryBasePath + "/test-chart:v0.1.0"},
					Exports:        nil,
				}
				rawProviderConfig, err := helper.ProviderConfigurationToRawExtension(config)
				Expect(err).NotTo(HaveOccurred())

				di = &lsv1alpha1.DeployItem{
					TypeMeta: metav1.TypeMeta{
						APIVersion: lsv1alpha1.SchemeGroupVersion.String(),
						Kind:       "DeployItem",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "private-registry-deploy-item",
						Namespace: "private-registry",
					},
					Spec: lsv1alpha1.DeployItemSpec{
						Timeout: &lsv1alpha1.Duration{Duration: 30 * time.Second},
						Target: &lsv1alpha1.ObjectReference{
							Name:      target.Name,
							Namespace: target.Namespace,
						},
						Type:          helm.Type,
						Configuration: rawProviderConfig,
					},
				}

				// create local copies of the variables since they are used multiple times and would be altered by
				// kubernetes during the create operation
				localns := *ns
				localtarget := *target
				localsecret := *secret
				locallsctx := *lsctx
				localdi := *di

				By("creating kubernetes objects")
				Expect(state.Create(ctx, &localns)).To(BeNil())
				Expect(state.Create(ctx, &localtarget)).To(BeNil())
				Expect(state.Create(ctx, &localsecret)).To(BeNil())
				Expect(state.Create(ctx, &locallsctx)).To(BeNil())
				Expect(state.Create(ctx, &localdi)).To(BeNil())

				By("update job id of deploy item to trigger reconciliation")
				Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
				Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())

				By("waiting for the deploy item to succeed")
				Expect(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhases.Succeeded, 1*time.Minute)).To(BeNil())
				cm := &corev1.ConfigMap{}
				Expect(state.Client.Get(ctx, kutil.ObjectKey("mychart-configmap", "private-registry"), cm)).To(Succeed())
				Expect(cm.Data["key"]).To(Equal("value"))

				By("removing the deploy item")
				utils.ExpectNoError(f.Client.Delete(ctx, di))
				// Set a new jobID to trigger a reconcile of the deploy item
				Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
				Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())

				utils.ExpectNoError(utils.WaitForObjectDeletion(ctx, f.Client, di, 2*time.Minute))

				Expect(state.CleanupState(ctx, envtest.WaitForDeletion(true))).To(BeNil())
			}

			It("deploy deploy item with chart in private registry (real helm deployer)", func() {
				testFunc(true)
			})

			It("deploy deploy item with chart in private registry (templating and manifest apply)", func() {
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
	err = lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhases.Succeeded, 1*time.Minute)

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
	utils.ExpectNoError(state.Client.Get(ctx, kutil.ObjectKey(chartName, state.Namespace), di))
	new_di := createHelmDeployItem(chartDir, valuesFile, chartName, target, nil, nil, useHelmDeployment)
	di.Spec = new_di.Spec

	utils.ExpectNoError(state.Update(ctx, di))
	// Set a new jobID to trigger a reconcile of the deploy item
	Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di)).To(Succeed())
	Expect(utils.UpdateJobIdForDeployItemC(ctx, state.Client, di, metav1.Now())).To(Succeed())
	By("Waiting for the DeployItem Update to succeed")
	utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, f.Client, di, lsv1alpha1.DeployItemPhases.Succeeded, 20*time.Second))
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
		DisableDefault: true,
		CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
			{
				Name: "valueIsCorrect",
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
		HelmDeployment: ptr.To[bool](useHelmDeployment),
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
			Timeout:       &lsv1alpha1.Duration{Duration: 30 * time.Second},
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
