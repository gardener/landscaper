// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"context"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	k8sv1 "k8s.io/api/core/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
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
			var (
				do1  = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				do2  = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				do3  = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			// Imports from DataObjects

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-1", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Imports from ConfigMaps

			By("Create ConfigMaps with import data")
			cm1 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm1, path.Join(testdataDir, "installation-2", "import-configmap-name.yaml")))
			cm1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, cm1))

			cm2 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm2, path.Join(testdataDir, "installation-2", "import-configmap-namespace.yaml")))
			cm2.SetNamespace(state.Namespace)
			cm2.Data["configmapNamespace"] = state.Namespace
			utils.ExpectNoError(state.Create(ctx, cm2))

			cm3 := &k8sv1.ConfigMap{}
			utils.ExpectNoError(utils.ReadResourceFromFile(cm3, path.Join(testdataDir, "installation-2", "import-configmap-data.yaml")))
			cm3.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, cm3))

			By("Update Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-2", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData = map[string]string{"foo": "bar", "key3": "value3", "key4": "value4"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Imports from secrets

			By("Create Secrets with import data")
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

			By("Update Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-3", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData = map[string]string{"foo": "bar", "key5": "value5", "key6": "value6"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))
		})

		It("should update the name of a DataObject and a DeployItem", func() {
			var (
				do1  = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				do2  = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				do3  = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			// Imports from DataObjects

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-1", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Rename DataObjects and change DeployItem

			By("Rename DataObjects with import data")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-8", "import-do-name.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-8", "import-do-namespace.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-8", "import-do-data.yaml")))

			By("Update Installation: changed DeployItem")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-8", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData = map[string]string{"fooUpdated": "barUpdated", "key7": "value7", "key8": "value8"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name-updated", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data-updated", expectedData))
		})
	})
}
