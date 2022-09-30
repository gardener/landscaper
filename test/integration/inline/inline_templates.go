// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"context"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func InlineTemplateTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "inline")
	)

	Describe("Inline DeployExecution and ExportExecution", func() {

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

		It("a Blueprint should use an inline DeployExecution and ExportExecution", func() {
			var (
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				inst        = &lsv1alpha1.Installation{}
			)

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-inline-base", "import-do-namespace.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installations with inline blueprint")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-base", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Delete Installations")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute))

			By("Check deletion of ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			err = f.Client.Get(ctx, configMapKey, configMap)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("a Blueprint should use an inline subinstallation", func() {
			var (
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				//doData      = &lsv1alpha1.DataObject{} // contains the data of the ConfigMap
				inst = &lsv1alpha1.Installation{}
			)

			// Create Installation with static subinstallations

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-inline-root-1", "import-do-namespace.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installations with inline blueprint")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-root-1", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Check exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-name", "cm-example-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-data", expectedData))

			// Update Installation using a blueprint with templated subinstallations

			By("Update Installation: templated subinstallations")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-root-2", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData = map[string]string{"foo": "bar", "key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-key1", expectedData))
			expectedData = map[string]string{"foo": "bar", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-key2", expectedData))

			By("Check exports")
			expectedData = map[string]string{"key1": "cm-example-key1-x", "key2": "cm-example-key2-x"}
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "export-do-names", expectedData))

			// Delete Installation

			By("Delete Installations")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute))

			By("Check deletion of deployed ConfigMaps")
			configmapNames := []string{"cm-example-key1-x", "cm-example-key2-x"}
			for _, cmName := range configmapNames {
				configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: cmName}
				configMap := &k8sv1.ConfigMap{}
				err = f.Client.Get(ctx, configMapKey, configMap)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		})
	})
}
