// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	deployercmd "github.com/gardener/landscaper/pkg/deployer/lib/cmd"

	"github.com/google/uuid"

	"github.com/gardener/landscaper/pkg/deployer/helm/realhelmdeployer"
	"github.com/gardener/landscaper/pkg/deployer/lib/resourcemanager"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmc "helm.sh/helm/v3/pkg/chart"
	helmr "helm.sh/helm/v3/pkg/release"
	helmt "helm.sh/helm/v3/pkg/time"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/helm/v1alpha1/helper"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/deployer/helm"
	lsutils "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/simplelogger"
	"github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "helm Test Suite")
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

var _ = Describe("Template", func() {
	It("should ignore non-kubernetes manifests that are valid yaml", func() {
		ctx := logging.NewContext(context.Background(), logging.Discard())

		kubeconfig, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
		Expect(err).ToNot(HaveOccurred())
		chartData, closer := utils.ReadChartFrom("./testdata/testchart")
		defer closer()
		helmConfig := &helmv1alpha1.ProviderConfiguration{}
		helmConfig.Kubeconfig = base64.StdEncoding.EncodeToString(kubeconfig)
		helmConfig.Chart.Archive = &helmv1alpha1.ArchiveAccess{
			Raw: base64.StdEncoding.EncodeToString(chartData),
		}
		helmConfig.Name = "foo"
		helmConfig.Namespace = "foo"
		providerConfig, err := helper.ProviderConfigurationToRawExtension(helmConfig)
		Expect(err).ToNot(HaveOccurred())

		item := &lsv1alpha1.DeployItem{}
		item.Spec.Configuration = providerConfig

		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = item.Namespace
		h, err := helm.New(helmv1alpha1.Configuration{}, testenv.Client, testenv.Client, item, nil, lsCtx, nil)
		Expect(err).ToNot(HaveOccurred())
		files, crds, _, _, err := h.Template(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(crds).To(HaveKey("testchart/crds/crontabs.yaml"))
		Expect(files).To(HaveKey("testchart/templates/secret.yaml"))
		Expect(files).To(HaveKey("testchart/templates/note.txt"))

		objects, err := kutil.ParseFiles(logging.Discard(), files)
		Expect(err).ToNot(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})

	Context("Integration", func() {

		var (
			ctx    context.Context
			cancel context.CancelFunc
			state  *envtest.State
			mgr    manager.Manager
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(logging.NewContextWithDiscard(context.Background()))
			var err error
			state, err = testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			mgr, err = manager.New(testenv.Env.Config, manager.Options{
				Scheme:             api.LandscaperScheme,
				MetricsBindAddress: "0",
				NewClient:          lsutils.NewUncachedClient(lsutils.LsResourceClientBurstDefault, lsutils.LsResourceClientQpsDefault),
			})
			Expect(err).ToNot(HaveOccurred())

			do := deployercmd.DefaultOptions{
				LsKubeconfig: "",
				Log:          logging.Wrap(simplelogger.NewIOLogger(GinkgoWriter)),
				LsMgr:        mgr,
				HostMgr:      mgr,
				LsClient:     nil,
				HostClient:   nil,
			}
			Expect(helm.AddDeployerToManager(&do, helmv1alpha1.Configuration{},
				"helmintegration"+utils.GetNextCounter())).To(Succeed())

			timeout.ActivateStandardTimeoutChecker()

			go func() {
				Expect(mgr.Start(ctx)).To(Succeed())
			}()
		})

		AfterEach(func() {
			defer cancel()
			Expect(state.CleanupState(ctx)).To(Succeed())
		})

		isFinished := func(item *lsv1alpha1.DeployItem) bool {
			if err := state.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item); err != nil {
				return false
			}
			return item.Status.JobIDFinished == item.Status.JobID
		}

		It("should create the release namespace if configured", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./chartresolver/testdata/testchart")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKey("some-namespace", ""), &corev1.Namespace{}); err != nil {
					return err
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "additional namespace should be created")
		})

		It("should export helm values", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./chartresolver/testdata/testchart")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmValues := map[string]interface{}{
				"MyKey": "SomeVal",
			}

			helmValuesRaw, err := json.Marshal(helmValues)
			Expect(err).ToNot(HaveOccurred())

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
				Values:          helmValuesRaw,
				Exports: &managedresource.Exports{
					Exports: []managedresource.Export{
						{
							Key:      "ExportA",
							JSONPath: ".Values.MyKey",
						},
					},
				},
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			export := &corev1.Secret{}

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, client.ObjectKeyFromObject(item), item); err != nil {
					return err
				}

				if item.Status.ExportReference == nil {
					return fmt.Errorf("export reference not found")
				}

				return nil
			}, 20*time.Second, 1*time.Second).Should(Succeed(), "export should be created")

			Expect(testenv.Client.Get(ctx, item.Status.ExportReference.NamespacedName(), export)).ToNot(HaveOccurred())
			Expect(export.Data).To(HaveKey("config"))
			configRaw := export.Data["config"]

			var config map[string]interface{}
			Expect(json.Unmarshal(configRaw, &config)).ToNot(HaveOccurred())
			Expect(config).To(HaveKeyWithValue("ExportA", "SomeVal"))
		})

		It("should deploy a chart with configmap lists", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart2")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item); err != nil {
					return err
				}
				if !item.Status.Phase.IsFinal() {
					return fmt.Errorf("deploy item is unfinished")
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should reach a final phase")

			Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))

			helmProviderStatus := &helmv1alpha1.ProviderStatus{}
			Expect(json.Unmarshal(item.Status.ProviderStatus.Raw, helmProviderStatus)).To(Succeed())
			Expect(helmProviderStatus.ManagedResources).To(HaveLen(0)) // would contain only resources that are relevant for the default readiness check
			cm := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-cm-1", "some-namespace"), cm)).To(Succeed())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-cm-2", "some-namespace"), cm)).To(Succeed())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-cm-3", "some-namespace"), cm)).To(Succeed())

			itemObjectKey := client.ObjectKeyFromObject(item)
			Expect(testenv.Client.Delete(ctx, item)).To(Succeed())
			Eventually(func() error {
				Expect(testenv.Client.Get(ctx, itemObjectKey, item)).To(Succeed())
				return utils.UpdateJobIdForDeployItem(ctx, testenv, item, metav1.Now())
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be updated with a new job id")

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, itemObjectKey, item); apierrors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("deploy item not yet deleted")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be deleted")
		})

		It("should deploy a chart with subchart", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart5")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Chart:           chartAccess,
				Name:            "test",
				Namespace:       "some-namespace",
				CreateNamespace: true,
				Values:          []byte(`{"subchart-enabled": true}`), // subchart enabled
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item); err != nil {
					return err
				}
				if !item.Status.Phase.IsFinal() {
					return fmt.Errorf("deploy item is unfinished")
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should reach a final phase")

			Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))

			helmProviderStatus := &helmv1alpha1.ProviderStatus{}
			Expect(json.Unmarshal(item.Status.ProviderStatus.Raw, helmProviderStatus)).To(Succeed())
			// There should be 2 configmaps, one from the chart and one from the subchart
			Expect(helmProviderStatus.ManagedResources).To(HaveLen(0))
			cm := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-chart-configmap", "some-namespace"), cm)).To(Succeed())
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-subchart-configmap", "some-namespace"), cm)).To(Succeed())

			itemObjectKey := client.ObjectKeyFromObject(item)
			Expect(testenv.Client.Delete(ctx, item)).To(Succeed())
			Eventually(func() error {
				Expect(testenv.Client.Get(ctx, itemObjectKey, item)).To(Succeed())
				return utils.UpdateJobIdForDeployItem(ctx, testenv, item, metav1.Now())
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be updated with a new job id")

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, itemObjectKey, item); apierrors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("deploy item not yet deleted")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be deleted")
		})

		It("should skip a disabled subchart", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart5")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Chart:           chartAccess,
				Name:            "test",
				Namespace:       "some-namespace",
				CreateNamespace: true,
				Values:          []byte(`{"subchart-enabled": false}`), // subchart disabled
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKeyFromObject(item), item); err != nil {
					return err
				}
				if !item.Status.Phase.IsFinal() {
					return fmt.Errorf("deploy item is unfinished")
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should reach a final phase")

			Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Succeeded))

			helmProviderStatus := &helmv1alpha1.ProviderStatus{}
			Expect(json.Unmarshal(item.Status.ProviderStatus.Raw, helmProviderStatus)).To(Succeed())
			// There should be only 1 configmap, the other one in the subchart should not have been deployed
			Expect(helmProviderStatus.ManagedResources).To(HaveLen(0))
			cm := &corev1.ConfigMap{}
			Expect(testenv.Client.Get(ctx, kutil.ObjectKey("test-chart-configmap", "some-namespace"), cm)).To(Succeed())

			itemObjectKey := client.ObjectKeyFromObject(item)
			Expect(testenv.Client.Delete(ctx, item)).To(Succeed())
			Eventually(func() error {
				Expect(testenv.Client.Get(ctx, itemObjectKey, item)).To(Succeed())
				return utils.UpdateJobIdForDeployItem(ctx, testenv, item, metav1.Now())
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be updated with a new job id")

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, itemObjectKey, item); apierrors.IsNotFound(err) {
					return nil
				}
				return fmt.Errorf("deploy item not yet deleted")
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "deploy item should be deleted")
		})

		It("should respect the custom readiness check timeout when set", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./chartresolver/testdata/testchart")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			requirementValue := map[string]string{
				"value": "true",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
				ReadinessChecks: readinesschecks.ReadinessCheckConfiguration{
					CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
						{
							Name: "my-check",
							Resource: []lsv1alpha1.TypedObjectReference{
								{
									APIVersion: "v1",
									Kind:       "ConfigMap",
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      "my-cm",
										Namespace: state.Namespace,
									},
								},
							},
							Requirements: []readinesschecks.RequirementSpec{
								{
									JsonPath: ".data.ready",
									Operator: selection.Equals,
									Value: []runtime.RawExtension{
										{
											Raw: requirementValueMarshaled,
										},
									},
								},
							},
						},
					},
				},
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			go func() {
				defer GinkgoRecover()
				time.Sleep(10 * time.Second)
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-cm",
						Namespace: state.Namespace,
					},
					Data: map[string]string{
						"ready": "true",
					},
				}
				Expect(state.Client.Create(ctx, cm)).To(Succeed())
			}()

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, client.ObjectKeyFromObject(item), item); err != nil {
					return err
				}

				if item.Status.Phase != lsv1alpha1.DeployItemPhases.Succeeded {
					return fmt.Errorf("deploy item phase is not succeeded")
				}

				return nil
			}, 30*time.Second, 1*time.Second).Should(Succeed(), "custom readiness checks fulfilled")
		})

		It("should fail with a timeout", func() {
			// This test creates a deploy item that deploys a helm chart with a single config map.
			// Its reconciliation should fail with a timeout after 1 second. The timeout should occur at the latest
			// during a custom readiness check that expects a wrong value.
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart3")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			requirementValue := map[string]string{
				"value": "value_WRONG",
			}
			requirementValueMarshaled, err := json.Marshal(requirementValue)
			Expect(err).ToNot(HaveOccurred())

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       "some-namespace",
				Chart:           chartAccess,
				CreateNamespace: true,
				ReadinessChecks: readinesschecks.ReadinessCheckConfiguration{
					CustomReadinessChecks: []readinesschecks.CustomReadinessCheckConfiguration{
						{
							Name: "my-check",
							Resource: []lsv1alpha1.TypedObjectReference{
								{
									APIVersion: "v1",
									Kind:       "ConfigMap",
									ObjectReference: lsv1alpha1.ObjectReference{
										Name:      "mychart-configmap",
										Namespace: "some-namespace",
									},
								},
							},
							Requirements: []readinesschecks.RequirementSpec{
								{
									JsonPath: ".data.key",
									Operator: selection.Equals,
									Value: []runtime.RawExtension{
										{
											Raw: requirementValueMarshaled,
										},
									},
								},
							},
						},
					},
				},
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				WithTimeout(1 * time.Second).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, client.ObjectKeyFromObject(item), item); err != nil {
					return err
				}

				if item.Status.Phase != lsv1alpha1.DeployItemPhases.Failed {
					return fmt.Errorf("deploy item phase is not failed")
				}

				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "custom readiness checks fulfilled")

			Expect(item.Status.LastError).NotTo(BeNil())
			Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		})

		It("should unblock a pending helm release and finally install the helm chart", func() {
			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			testNamespaceName := "test4"

			helmRelease := helmr.Release{
				Name: "test",
				Info: &helmr.Info{
					FirstDeployed: helmt.Now(),
					LastDeployed:  helmt.Now(),
					Description:   "Pending Installation",
					Status:        helmr.StatusPendingInstall,
				},
				Chart: &helmc.Chart{
					Metadata: &helmc.Metadata{
						Name:        "testchart",
						Version:     "0.1.0",
						Description: "A Helm chart for Kubernetes",
						APIVersion:  "v2",
						AppVersion:  "1.16.0",
						Type:        "application",
					},
					Values: map[string]interface{}{
						"replicaCount": 1,
					},
				},
				Version:   1,
				Namespace: testNamespaceName,
			}

			helmReleaseMarshaled, err := json.Marshal(helmRelease)
			Expect(err).ToNot(HaveOccurred())

			var buf bytes.Buffer
			w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			Expect(err).ToNot(HaveOccurred())
			_, err = w.Write(helmReleaseMarshaled)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Close()).To(Succeed())

			helmReleaseEncoded := base64.StdEncoding.EncodeToString(buf.Bytes())

			helmReleaseSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sh.helm.release.v1.test.v1",
					Namespace: testNamespaceName,
					Labels: map[string]string{
						"name":    "test",
						"owner":   "helm",
						"status":  "pending-install",
						"version": "1",
					},
				},
				StringData: map[string]string{
					"release": helmReleaseEncoded,
				},
				Type: "helm.sh/release.v1",
			}

			testNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespaceName,
				},
			}

			Expect(testenv.Client.Create(ctx, testNamespace)).To(Succeed())
			Expect(testenv.Client.Create(ctx, helmReleaseSecret)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart4")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Name:            "test",
				Namespace:       testNamespaceName,
				Chart:           chartAccess,
				CreateNamespace: false,
				HelmDeployment:  pointer.Bool(true),
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "myitem").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(func() error {
				if err := testenv.Client.Get(ctx, kutil.ObjectKey("mysecret", testNamespaceName), &corev1.Secret{}); err != nil {
					return err
				}
				return nil
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "additional namespace should be created")
		})

		It("should time out at checkpoints of the real helm deployer", func() {
			// This test creates/reconciles/deletes a real helm deploy item. Before these operations,
			// it replaces the standard timeout checker by test implementations that throw a timeout error at certain
			// check points. It verifies that the expected timeouts actually occur.

			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart3")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Chart:           chartAccess,
				Name:            "test",
				Namespace:       "some-namespace",
				CreateNamespace: true,
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "helm-timeout-test").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())

			timeout.ActivateCheckpointTimeoutChecker(helm.TimeoutCheckpointHelmStartReconcile)
			defer timeout.ActivateStandardTimeoutChecker()

			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), "deploy item should eventually have a final phase")
			Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
			Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
			Expect(item.Status.LastError).NotTo(BeNil())
			Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
			Expect(item.Status.LastError.Message).To(ContainSubstring(helm.TimeoutCheckpointHelmStartReconcile))

			for _, checkpoint := range []string{
				helm.TimeoutCheckpointHelmStartProgressing,
				helm.TimeoutCheckpointHelmStartApplyFiles,
				realhelmdeployer.TimeoutCheckpointHelmBeforeInstallingRelease,
				helm.TimeoutCheckpointHelmBeforeReadinessCheck,
				helm.TimeoutCheckpointHelmDefaultReadinessChecks,
				helm.TimeoutCheckpointHelmBeforeReadingExportValues,
			} {
				timeout.ActivateCheckpointTimeoutChecker(checkpoint)
				item.Status.SetJobID(uuid.New().String())
				Expect(state.Client.Status().Update(ctx, item)).To(Succeed())

				description := fmt.Sprintf("deploy item should fail with timeout at checkpoint %s", checkpoint)
				Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), description)
				Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed(), description)
				Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed), description)
				Expect(item.Status.LastError).NotTo(BeNil(), description)
				Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout), description)
				Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint), description)
			}

			Expect(state.Client.Delete(ctx, item)).To(Succeed())
			Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())

			for _, checkpoint := range []string{
				helm.TimeoutCheckpointHelmStartDelete,
				helm.TimeoutCheckpointHelmStartDeleting,
				realhelmdeployer.TimeoutCheckpointHelmBeforeDeletingRelease,
			} {
				timeout.ActivateCheckpointTimeoutChecker(checkpoint)
				item.Status.SetJobID(uuid.New().String())
				Expect(state.Client.Status().Update(ctx, item)).To(Succeed())

				description := fmt.Sprintf("deploy item should fail with timeout at checkpoint %s", checkpoint)
				Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), description)
				Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed(), description)
				Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.DeleteFailed), description)
				Expect(item.Status.LastError).NotTo(BeNil(), description)
				Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout), description)
				Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint), description)
			}
		})

		It("should time out at checkpoints of the manifest helm deployer", func() {
			// This test creates/reconciles/deletes a manifest helm deploy item. Before these operations,
			// it replaces the standard timeout checker by test implementations that throw a timeout error at certain
			// check points. It verifies that the expected timeouts actually occur.

			Expect(utils.CreateExampleDefaultContext(ctx, testenv.Client, state.Namespace)).To(Succeed())
			target, err := utils.CreateKubernetesTarget(state.Namespace, "my-target", testenv.Env.Config)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.Create(ctx, target)).To(Succeed())

			chartBytes, closer := utils.ReadChartFrom("./testdata/testchart3")
			defer closer()

			chartAccess := helmv1alpha1.Chart{
				Archive: &helmv1alpha1.ArchiveAccess{
					Raw: base64.StdEncoding.EncodeToString(chartBytes),
				},
			}

			helmConfig := &helmv1alpha1.ProviderConfiguration{
				Chart:           chartAccess,
				Name:            "test",
				Namespace:       "some-namespace",
				CreateNamespace: true,
				HelmDeployment:  pointer.Bool(false),
			}
			item, err := helm.NewDeployItemBuilder().
				Key(state.Namespace, "helm-timeout-test-2").
				ProviderConfig(helmConfig).
				Target(target.Namespace, target.Name).
				GenerateJobID().
				Build()
			Expect(err).ToNot(HaveOccurred())

			timeout.ActivateCheckpointTimeoutChecker(helm.TimeoutCheckpointHelmStartReconcile)
			defer timeout.ActivateStandardTimeoutChecker()

			Expect(state.Create(ctx, item, envtest.UpdateStatus(true))).To(Succeed())

			Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), "deploy item should eventually have a final phase")
			Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())
			Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed))
			Expect(item.Status.LastError).NotTo(BeNil())
			Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
			Expect(item.Status.LastError.Message).To(ContainSubstring(helm.TimeoutCheckpointHelmStartReconcile))

			for _, checkpoint := range []string{
				helm.TimeoutCheckpointHelmStartProgressing,
				helm.TimeoutCheckpointHelmStartApplyFiles,
				helm.TimeoutCheckpointHelmStartCreateManifests,
				helm.TimeoutCheckpointHelmStartApplyManifests,
				resourcemanager.TimeoutCheckpointDeployerProcessManagedResourceManifests,
				resourcemanager.TimeoutCheckpointDeployerProcessManifests,
				resourcemanager.TimeoutCheckpointDeployerApplyManifests,
				helm.TimeoutCheckpointHelmBeforeReadinessCheck,
			} {
				timeout.ActivateCheckpointTimeoutChecker(checkpoint)
				item.Status.SetJobID(uuid.New().String())
				Expect(state.Client.Status().Update(ctx, item)).To(Succeed())

				description := fmt.Sprintf("deploy item should fail with timeout at checkpoint %s", checkpoint)
				Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), description)
				Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed(), description)
				Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.Failed), description)
				Expect(item.Status.LastError).NotTo(BeNil(), description)
				Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout), description)
				Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint), description)
			}

			Expect(state.Client.Delete(ctx, item)).To(Succeed())
			Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed())

			for _, checkpoint := range []string{
				helm.TimeoutCheckpointHelmStartDelete,
				helm.TimeoutCheckpointHelmStartDeleting,
				helm.TimeoutCheckpointHelmDeleteResources,
			} {
				timeout.ActivateCheckpointTimeoutChecker(checkpoint)
				item.Status.SetJobID(uuid.New().String())
				Expect(state.Client.Status().Update(ctx, item)).To(Succeed())

				description := fmt.Sprintf("deploy item should fail with timeout at checkpoint %s", checkpoint)
				Eventually(isFinished, 10*time.Second, 1*time.Second).WithArguments(item).Should(BeTrue(), description)
				Expect(state.Client.Get(ctx, client.ObjectKeyFromObject(item), item)).To(Succeed(), description)
				Expect(item.Status.Phase).To(Equal(lsv1alpha1.DeployItemPhases.DeleteFailed), description)
				Expect(item.Status.LastError).NotTo(BeNil(), description)
				Expect(item.Status.LastError.Codes).To(ContainElement(lsv1alpha1.ErrorTimeout), description)
				Expect(item.Status.LastError.Message).To(ContainSubstring(checkpoint), description)
			}
		})

	})

})
