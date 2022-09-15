// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"context"
	"path"
	"path/filepath"
	"time"

	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ImportExportTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "import-export")
	)

	Describe("Imports and Exports", func() {

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

		It("should read imports from DataObjects, ConfigMaps and Secrets", func() {
			// Imports from dataobjects

			By("Create 1st data object with import data")
			do1 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			do1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do1))

			By("Create 2nd data object with import data")
			do2 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do2, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			do2.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do2, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do2))

			By("Create 3rd data object with import data")
			do3 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do3, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))
			do3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do3))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key1", "value1"))
			Expect(configMap.Data).To(HaveKeyWithValue("key2", "value2"))

			By("Check export dataobjects")
			exportDo := &lsv1alpha1.DataObject{}
			exportDoKey := client.ObjectKey{Name: "export-do-name", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportName := ""
			utils.GetDataObjectData(exportDo, &exportName)
			Expect(exportName).To(Equal("cm-example-x"))

			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-data", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			exportMap := map[string]string{}
			utils.GetDataObjectData(exportDo, &exportMap)
			Expect(exportMap).To(HaveLen(3))
			Expect(exportMap).To(HaveKeyWithValue("foo", "bar"))
			Expect(exportMap).To(HaveKeyWithValue("key1", "value1"))
			Expect(exportMap).To(HaveKeyWithValue("key2", "value2"))

			// Imports from configmaps

			By("Create 1st configmap with import data")
			cm1 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm1, path.Join(testdataDir, "installation-2", "import-configmap-name.yaml")))
			cm1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, cm1))

			By("Create 2nd configmap with import data")
			cm2 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm2, path.Join(testdataDir, "installation-2", "import-configmap-namespace.yaml")))
			cm2.SetNamespace(state.Namespace)
			cm2.Data["configmapNamespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm2))

			By("Create 3rd configmap with import data")
			cm3 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm3, path.Join(testdataDir, "installation-2", "import-configmap-data.yaml")))
			cm3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, cm3))

			By("Create installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-2", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key3", "value3"))
			Expect(configMap.Data).To(HaveKeyWithValue("key4", "value4"))

			By("Check export dataobject")
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
			Expect(exportMap).To(HaveKeyWithValue("key3", "value3"))
			Expect(exportMap).To(HaveKeyWithValue("key4", "value4"))

			// Imports from secrets

			By("Create 1st secret with import data")
			secret1 := &k8sv1.Secret{}
			utils.ExpectNoError(utils.ReadResourceFromFile(secret1, path.Join(testdataDir, "installation-3", "import-secret-name.yaml")))
			secret1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, secret1))

			By("Create 2nd secret with import data")
			secret2 := &k8sv1.Secret{}
			utils.ExpectNoError(utils.ReadResourceFromFile(secret2, path.Join(testdataDir, "installation-3", "import-secret-namespace.yaml")))
			secret2.SetNamespace(state.Namespace)
			secret2.StringData["configmapNamespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, secret2))

			By("Create 3rd secret with import data")
			secret3 := &k8sv1.Secret{}
			utils.ExpectNoError(utils.ReadResourceFromFile(secret3, path.Join(testdataDir, "installation-3", "import-secret-data.yaml")))
			secret3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, secret3))

			By("Create installation")
			instOld = &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-3", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(3))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
			Expect(configMap.Data).To(HaveKeyWithValue("key5", "value5"))
			Expect(configMap.Data).To(HaveKeyWithValue("key6", "value6"))

			By("Check export dataobject")
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
			Expect(exportMap).To(HaveKeyWithValue("key5", "value5"))
			Expect(exportMap).To(HaveKeyWithValue("key6", "value6"))
		})
	})
}
