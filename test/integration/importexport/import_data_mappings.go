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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
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
			var (
				do1  = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				do2  = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				do3  = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-4", "import-do-data-1.yaml")))
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-4", "import-do-data-2.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-4", "import-do-namespace.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-4", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key7": "value7", "key8": "value8"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))
		})
	})
}
