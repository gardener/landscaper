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

func ImportDataMappingsTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "import-export")
	)

	Describe("Import Data Mappings", func() {

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

		It("should apply import data mappings", func() {
			By("Create 1st data object with import data")
			do1 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-4", "import-do-data-1.yaml")))
			do1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do1))

			By("Create 3rd data object with import data")
			do2 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do2, path.Join(testdataDir, "installation-4", "import-do-data-2.yaml")))
			do2.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do2))

			By("Create 1st data object with import data")
			do3 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do3, path.Join(testdataDir, "installation-4", "import-do-namespace.yaml")))
			do3.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do3, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do3))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-4", "installation.yaml")))
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
			Expect(configMap.Data).To(HaveKeyWithValue("key7", "value7"))
			Expect(configMap.Data).To(HaveKeyWithValue("key8", "value8"))

			By("Check export dataobject")
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
			Expect(exportMap).To(HaveKeyWithValue("key7", "value7"))
			Expect(exportMap).To(HaveKeyWithValue("key8", "value8"))
		})
	})
}
