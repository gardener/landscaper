package subinstallations

import (
	"context"
	"path"
	"path/filepath"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func SubinstallationTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "subinstallations")
	)

	Describe("Subinstallations", func() {

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

		It("should add and remove subinstallations", func() {
			By("Create DataObjects with imports")

			do1 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			do1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do1))

			do2 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do2, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			do2.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do2, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do2))

			do3 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do3, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))
			do3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do3))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation referencing component version v0.2.0 with two subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation-v0.2.0.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			By("Check DataObjects with exports")
			exportDo := &lsv1alpha1.DataObject{}
			exportDoKey := client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName := ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-x-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap := map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(3))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))
			Expect(exportMap).To(HaveKeyWithValue("key2", "value2"))

			// Add subinstallation

			By("Update installation referencing component version v0.3.0 with three subinstallations")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation-v0.3.0.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			By("Check DataObjects with exports")
			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName = ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-x-x-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap = map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(3))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))
			Expect(exportMap).To(HaveKeyWithValue("key2", "value2"))

			// Remove subinstallations

			By("Update installation referencing component version v0.1.0 with one subinstallation")
			instOld = &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation-v0.1.0.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			By("Check deleted ConfigMaps")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x"}
			configMap = &k8sv1.ConfigMap{}
			Expect(f.Client.Get(ctx, configMapKey, configMap)).To(HaveOccurred())

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x-x"}
			configMap = &k8sv1.ConfigMap{}
			Expect(f.Client.Get(ctx, configMapKey, configMap)).To(HaveOccurred())

			By("Check DataObjects with exports")
			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName = ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap = map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(3))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))
			Expect(exportMap).To(HaveKeyWithValue("key2", "value2"))
		})

		It("should update the version of subinstallations", func() {
			By("Create DataObjects with imports")

			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-2", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with blueprint versions: v0.3.0 (root), v0.1.0, v0.1.0, v0.1.0")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-2", "installation-v0.3.0.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			By("Check DataObjects with exports")
			exportDo := &lsv1alpha1.DataObject{}
			exportDoKey := client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName := ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-x-x-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap := map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(2))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))

			// Update installation

			By("Update installation with blueprint versions: v0.4.0 (root), v0.2.0, v0.1.0, v0.2.0")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-2", "installation-v0.4.0.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-y"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-y-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			By("Check DataObjects with exports")
			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName = ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-y-x-y"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap = map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(2))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))
		})

		It("should update imports and exports", func() {
			By("Create DataObjects with imports")

			do1 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-3", "import-do-name-1.yaml")))
			do1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do1))

			do2 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do2, path.Join(testdataDir, "installation-3", "import-do-namespace.yaml")))
			do2.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do2, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do2))

			do3 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do3, path.Join(testdataDir, "installation-3", "import-do-data-1.yaml")))
			do3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do3))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-3", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-1"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-1-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-1-x-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))

			By("Check DataObjects with exports")
			exportDo := &lsv1alpha1.DataObject{}
			exportDoKey := client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName := ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-1-x-x-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap := map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(2))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))

			// Update imports

			By("Update DataObjects with imports")
			do1Old := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(do1), do1Old))
			do1 = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-3", "import-do-name-2.yaml")))
			do1.SetNamespace(state.Namespace)
			do1.ObjectMeta.ResourceVersion = do1Old.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, do1))

			do3Old := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(do3), do3Old))
			do3 = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do3, path.Join(testdataDir, "installation-3", "import-do-data-2.yaml")))
			do3.SetNamespace(state.Namespace)
			do3.ObjectMeta.ResourceVersion = do3Old.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, do3))

			By("Reconcile Installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-3", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-2"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-2-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-2-x-x"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			By("Check DataObjects with exports")
			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName = ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-2-x-x-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap = map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(2))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key2", "value2"))
		})
	})
}
