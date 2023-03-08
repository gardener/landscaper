// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/sys/unix"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/container"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/utils"

	"github.com/gardener/landscaper/test/framework"
)

// ContainerTests implemets tests for the Landscaper container deployer.
func ContainerTests(f *framework.Framework) {
	var (
		state       = f.Register()
		ctx         context.Context
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "container-deployer")
		doName      *lsv1alpha1.DataObject
		doNamespace *lsv1alpha1.DataObject
		doData      *lsv1alpha1.DataObject

		createTargetAndDataObjects = func() {
			By("create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("create name data object")
			doName = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(doName, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			doName.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, doName))

			By("create namespace data object")
			doNamespace = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(doNamespace, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			doNamespace.SetNamespace(state.Namespace)
			utils.SetDataObjectData(doNamespace, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, doNamespace))

			By("create configmap data data object")
			doData = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(doData, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))
			doData.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, doData))
		}
	)
	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		ctx.Done()
	})

	It("should create the configmap", func() {
		createTargetAndDataObjects()

		By("create installation")
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
		inst.SetNamespace(state.Namespace)
		utils.ExpectNoError(state.Create(ctx, inst))

		By("Wait for installation to finish")
		utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 5*time.Minute))

		var (
			configmapName string
			configmapData map[string]string
		)
		utils.ExpectNoError(json.Unmarshal(doName.Data.RawMessage, &configmapName))
		utils.ExpectNoError(json.Unmarshal(doData.Data.RawMessage, &configmapData))

		configmap := &v1.ConfigMap{}
		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: configmapName, Namespace: state.Namespace}, configmap))

		Expect(configmap.Name).To(Equal(configmapName))
		Expect(configmap.Namespace).To(Equal(state.Namespace))
		Expect(configmap.Data).To(Equal(configmapData))
	})

	It("should create the exports", func() {
		createTargetAndDataObjects()

		By("create installation")
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
		inst.SetNamespace(state.Namespace)
		utils.ExpectNoError(state.Create(ctx, inst))

		By("Wait for installation to finish")
		utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 5*time.Minute))

		var (
			configmapDataExpected map[string]string
			configmapDataActual   map[string]string
		)
		utils.ExpectNoError(json.Unmarshal(doData.Data.RawMessage, &configmapDataExpected))

		configmapDataExport := &lsv1alpha1.DataObject{}
		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: "configmapdata", Namespace: state.Namespace}, configmapDataExport))
		utils.ExpectNoError(json.Unmarshal(configmapDataExport.Data.RawMessage, &configmapDataActual))
		Expect(configmapDataExpected).To(Equal(configmapDataActual))

		var componentData map[string]string
		componentDataExport := &lsv1alpha1.DataObject{}
		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: "component", Namespace: state.Namespace}, componentDataExport))
		utils.ExpectNoError(json.Unmarshal(componentDataExport.Data.RawMessage, &componentData))
		Expect(componentData).To(HaveKey("name"))
		Expect(componentData["name"]).To(Equal(inst.Spec.ComponentDescriptor.Reference.ComponentName))
		Expect(componentData).To(HaveKey("version"))
		Expect(componentData["version"]).To(Equal(inst.Spec.ComponentDescriptor.Reference.Version))

		var (
			contentData []map[string]interface{}
		)
		contentDataExport := &lsv1alpha1.DataObject{}
		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: "content", Namespace: state.Namespace}, contentDataExport))
		utils.ExpectNoError(json.Unmarshal(contentDataExport.Data.RawMessage, &contentData))
		Expect(contentData).To(HaveLen(4))

		getFile := func(name string) map[string]interface{} {
			for _, content := range contentData {
				if content["name"] == name {
					return content["stat"].(map[string]interface{})
				}
			}
			return nil
		}

		verifyFile := func(file map[string]interface{}) {
			Expect(file).ToNot(BeNil())
			Expect(file["uid"]).To(BeEquivalentTo(1000))
			Expect(file["gid"]).To(BeEquivalentTo(2000))
			blueprintMode := file["mode"].(int64)
			Expect((blueprintMode & unix.S_IWUSR) > 0).To(BeTrue())
			Expect((blueprintMode & unix.S_IRUSR) > 0).To(BeTrue())
		}

		blueprint := getFile("blueprint.yaml")
		verifyFile(blueprint)
		deployExecution := getFile("deploy-execution.yaml")
		verifyFile(deployExecution)
		exportExecution := getFile("export-execution.yaml")
		verifyFile(exportExecution)
		script := getFile("script.py")
		verifyFile(script)
	})

	It("should delete the configmap", func() {
		createTargetAndDataObjects()

		By("create installation")
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
		inst.SetNamespace(state.Namespace)
		utils.ExpectNoError(state.Create(ctx, inst))

		By("Wait for installation to finish")
		utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 5*time.Minute))

		var configmapName string
		utils.ExpectNoError(json.Unmarshal(doName.Data.RawMessage, &configmapName))

		configmap := &v1.ConfigMap{}
		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: configmapName, Namespace: state.Namespace}, configmap))

		utils.ExpectNoError(state.Client.Delete(ctx, inst))

		err := wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
			if err1 := state.GetWithRetry(ctx, client.ObjectKeyFromObject(inst), inst); err1 != nil {
				if k8serrors.IsNotFound(err1) {
					return true, nil
				} else {
					return false, err1
				}
			}
			return false, nil
		})
		utils.ExpectNoError(err)
	})

	It("should read and update state", func() {
		createTargetAndDataObjects()

		By("create installation")
		inst := &lsv1alpha1.Installation{}
		utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
		inst.SetNamespace(state.Namespace)
		utils.ExpectNoError(state.Create(ctx, inst))

		By("Wait for installation to finish")
		utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 5*time.Minute))

		stateExport := &lsv1alpha1.DataObject{}
		var stateData map[string]interface{}

		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: "state", Namespace: state.Namespace}, stateExport))
		utils.ExpectNoError(json.Unmarshal(stateExport.Data.RawMessage, &stateData))
		Expect(stateData).To(HaveKey("count"))
		Expect(stateData["count"]).To(BeEquivalentTo(1))

		By("Reconciling the installation")
		utils.ExpectNoError(state.GetWithRetry(ctx, client.ObjectKeyFromObject(inst), inst))
		if inst.ObjectMeta.Annotations == nil {
			inst.ObjectMeta.Annotations = make(map[string]string)
		}
		inst.ObjectMeta.Annotations[lsv1alpha1.OperationAnnotation] = string(lsv1alpha1.ReconcileOperation)
		utils.ExpectNoError(state.Client.Update(ctx, inst))

		By("Wait for installation to finish")
		utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 5*time.Minute))

		utils.ExpectNoError(state.GetWithRetry(ctx, types.NamespacedName{Name: "state", Namespace: state.Namespace}, stateExport))
		utils.ExpectNoError(json.Unmarshal(stateExport.Data.RawMessage, &stateData))
		Expect(stateData).To(HaveKey("count"))
		Expect(stateData["count"]).To(BeEquivalentTo(2))
	})

	Context("Targets", func() {

		var (
			cdi                           *lsv1alpha1.DeployItem
			targetName                    = "test-target"
			secretName                    = "test-secret"
			targetContent                 = []byte(`{"foo": "bar"}`)
			expectedTargetContentAsObject interface{}
		)

		BeforeEach(func() {
			utils.ExpectNoError(json.Unmarshal(targetContent, &expectedTargetContentAsObject))
			conf := containerv1alpha1.ProviderConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "container.deployer.landscaper.gardener.cloud/v1alpha1",
					Kind:       "ProviderConfiguration",
				},
				Image:   "alpine",
				Command: []string{"/bin/sh"},
				Args:    []string{"-c", "cp $TARGET_PATH $EXPORTS_PATH"},
			}
			rawConf, err := json.Marshal(conf)
			utils.ExpectNoError(err)

			cdi = &lsv1alpha1.DeployItem{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    state.Namespace,
				},
				Spec: lsv1alpha1.DeployItemSpec{
					Type: container.Type,
					Target: &lsv1alpha1.ObjectReference{
						Name:      targetName,
						Namespace: state.Namespace,
					},
					Configuration: &runtime.RawExtension{
						Raw: rawConf,
					},
				},
			}
			lsv1alpha1helper.SetOperation(&cdi.ObjectMeta, lsv1alpha1.TestReconcileOperation)
		})

		It("should make the Target content available inside the container - inline configuration", func() {
			target := &lsv1alpha1.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetName,
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.TargetSpec{
					Type:          "landscaper.gardener.cloud/test",
					Configuration: lsv1alpha1.NewAnyJSONPointer(targetContent),
				},
			}
			utils.ExpectNoError(state.Create(ctx, target))
			utils.ExpectNoError(state.GetWithRetry(ctx, client.ObjectKeyFromObject(target), target))

			utils.ExpectNoError(state.Create(ctx, cdi))
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, state.Client, cdi, lsv1alpha1.DeployerPhases.Succeeded, 3*time.Minute))

			exportSecret := &v1.Secret{}
			utils.ExpectNoError(state.GetWithRetry(ctx, cdi.Status.ExportReference.NamespacedName(), exportSecret))
			Expect(exportSecret.Data).To(HaveKey(lsv1alpha1.DataObjectSecretDataKey))
			rawResolvedTarget := exportSecret.Data[lsv1alpha1.DataObjectSecretDataKey]
			rt := &lsv1alpha1.ResolvedTarget{}
			utils.ExpectNoError(json.Unmarshal(rawResolvedTarget, rt))
			Expect(rt.Target).To(Equal(target))
			var actualTargetContentAsObject interface{}
			utils.ExpectNoError(json.Unmarshal([]byte(rt.Content), &actualTargetContentAsObject))
			Expect(actualTargetContentAsObject).To(Equal(expectedTargetContentAsObject))
		})

		It("should make the Target content available inside the container - secretRef with key", func() {
			secretKey := "key"
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: state.Namespace,
				},
				Data: map[string][]byte{
					secretKey: targetContent,
				},
			}
			utils.ExpectNoError(state.Create(ctx, secret))
			target := &lsv1alpha1.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetName,
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.TargetSpec{
					Type: "landscaper.gardener.cloud/test",
					SecretRef: &lsv1alpha1.LocalSecretReference{
						Name: secretName,
						Key:  secretKey,
					},
				},
			}
			utils.ExpectNoError(state.Create(ctx, target))
			utils.ExpectNoError(state.GetWithRetry(ctx, client.ObjectKeyFromObject(target), target))

			utils.ExpectNoError(state.Create(ctx, cdi))
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, state.Client, cdi, lsv1alpha1.DeployerPhases.Succeeded, 3*time.Minute))

			exportSecret := &v1.Secret{}
			utils.ExpectNoError(state.GetWithRetry(ctx, cdi.Status.ExportReference.NamespacedName(), exportSecret))
			Expect(exportSecret.Data).To(HaveKey(lsv1alpha1.DataObjectSecretDataKey))
			rawResolvedTarget := exportSecret.Data[lsv1alpha1.DataObjectSecretDataKey]
			rt := &lsv1alpha1.ResolvedTarget{}
			utils.ExpectNoError(json.Unmarshal(rawResolvedTarget, rt))
			Expect(rt.Target).To(Equal(target))
			var actualTargetContentAsObject interface{}
			utils.ExpectNoError(json.Unmarshal([]byte(rt.Content), &actualTargetContentAsObject))
			Expect(actualTargetContentAsObject).To(Equal(expectedTargetContentAsObject))
		})

		It("should make the Target content available inside the container - secretRef without key", func() {
			targetContentAsStringMap := map[string]string{}
			utils.ExpectNoError(json.Unmarshal(targetContent, &targetContentAsStringMap))
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: state.Namespace,
				},
				Data: map[string][]byte{},
			}
			for k, v := range targetContentAsStringMap {
				secret.Data[k] = []byte(v)
			}
			utils.ExpectNoError(state.Create(ctx, secret))
			target := &lsv1alpha1.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetName,
					Namespace: state.Namespace,
				},
				Spec: lsv1alpha1.TargetSpec{
					Type: "landscaper.gardener.cloud/test",
					SecretRef: &lsv1alpha1.LocalSecretReference{
						Name: secretName,
					},
				},
			}
			utils.ExpectNoError(state.Create(ctx, target))
			utils.ExpectNoError(state.GetWithRetry(ctx, client.ObjectKeyFromObject(target), target))

			utils.ExpectNoError(state.Create(ctx, cdi))
			utils.ExpectNoError(lsutils.WaitForDeployItemToFinish(ctx, state.Client, cdi, lsv1alpha1.DeployerPhases.Succeeded, 3*time.Minute))

			exportSecret := &v1.Secret{}
			utils.ExpectNoError(state.GetWithRetry(ctx, cdi.Status.ExportReference.NamespacedName(), exportSecret))
			Expect(exportSecret.Data).To(HaveKey(lsv1alpha1.DataObjectSecretDataKey))
			rawResolvedTarget := exportSecret.Data[lsv1alpha1.DataObjectSecretDataKey]
			rt := &lsv1alpha1.ResolvedTarget{}
			utils.ExpectNoError(json.Unmarshal(rawResolvedTarget, rt))
			Expect(rt.Target).To(Equal(target))
			var actualTargetContentAsObject interface{}
			utils.ExpectNoError(json.Unmarshal([]byte(rt.Content), &actualTargetContentAsObject))
			Expect(actualTargetContentAsObject).To(Equal(expectedTargetContentAsObject))
		})

	})
}
