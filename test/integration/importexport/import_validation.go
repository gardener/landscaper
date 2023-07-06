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

func ImportValidationTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "import-export")
	)

	Describe("Imports Validation", func() {

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

		It("an installation should fail if a required import is missing", func() {
			var (
				doName      = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				inst        = &lsv1alpha1.Installation{}
			)

			By("Create import DataObject containing the ConfigMap namespace")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-5", "import-do-namespace.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-5", "installation-neg.yaml")))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))

			// Fix the installation by adding the missing import

			By("Create import DataObject containing the ConfigMap name")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, doName, path.Join(testdataDir, "installation-5", "import-do-name.yaml")))

			By("Update Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-5", "installation-pos.yaml")))

			By("Wait for Installation to succeed")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
		})

		It("an installation should fail if an import value has the wrong data type", func() {
			var (
				doName      = &lsv1alpha1.DataObject{} // contains the name of the ConfigMap to be deployed by the Installation
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap
				inst        = &lsv1alpha1.Installation{}
			)

			By("Create import DataObjects containing the ConfigMap name and namespace")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, doName, path.Join(testdataDir, "installation-6", "import-do-name-neg.yaml")))
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-6", "import-do-namespace.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-6", "installation.yaml")))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))

			// Fix the type of import value

			By("Update import DataObject containing the ConfigMap name")
			utils.ExpectNoError(utils.UpdateDataObjectFromFile(ctx, state.State, doName, path.Join(testdataDir, "installation-6", "import-do-name-pos.yaml")))

			By("Trigger a new reconciliation of the Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-6", "installation.yaml")))

			By("Wait for Installation to succeed")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
		})
	})
}
