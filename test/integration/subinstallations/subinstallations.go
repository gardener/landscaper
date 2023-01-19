// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"context"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
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
			var (
				do1  = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				do2  = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				do3  = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-1", "import-do-name.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-1", "import-do-namespace.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-1", "import-do-data.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation referencing component version v0.2.0 with two subinstallations")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-1", "installation-v0.2.0.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Add subinstallation

			By("Update installation referencing component version v0.3.0 with three subinstallations")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-1", "installation-v0.3.0.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Remove subinstallations

			By("Update installation referencing component version v0.1.0 with one subinstallation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-1", "installation-v0.1.0.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check deleted ConfigMaps")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x"}
			configMap := &k8sv1.ConfigMap{}
			Expect(f.Client.Get(ctx, configMapKey, configMap)).To(HaveOccurred())

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-example-x-x"}
			configMap = &k8sv1.ConfigMap{}
			Expect(f.Client.Get(ctx, configMapKey, configMap)).To(HaveOccurred())

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))
		})

		It("should update the version of subinstallations", func() {
			var (
				do   = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				inst = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do, path.Join(testdataDir, "installation-2", "import-do-namespace.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with blueprint versions: v0.3.0 (root), v0.1.0, v0.1.0, v0.1.0")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-2", "installation-v0.3.0.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData := map[string]string{"foo": "bar", "key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Update installation

			By("Update installation with blueprint versions: v0.4.0 (root), v0.2.0, v0.1.0, v0.2.0")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-2", "installation-v0.4.0.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData = map[string]string{"foo": "bar", "key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-y", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-y-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-y-x-y"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))
		})

		It("should update imports and exports", func() {
			var (
				do1  = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				do2  = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				do3  = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-3", "import-do-name-1.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-3", "import-do-namespace.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-3", "import-do-data-1.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-3", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData := map[string]string{"foo": "bar", "key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-1", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-1-x", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-1-x-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-1-x-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Update imports

			By("Update DataObjects with imports")
			utils.ExpectNoError(utils.UpdateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-3", "import-do-name-2.yaml")))
			utils.ExpectNoError(utils.UpdateDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-3", "import-do-data-2.yaml")))

			By("Reconcile Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-3", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData = map[string]string{"foo": "bar", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-2", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-2-x", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-2-x-x", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-2-x-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))
		})
	})
}
