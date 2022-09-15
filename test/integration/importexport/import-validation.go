// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"context"
	"path"
	"path/filepath"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
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
			By("Create import DataObject containing the ConfigMap namespace")
			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-5", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-5", "installation-neg.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseFailed, 2*time.Minute))

			// Fix the installation by adding the missing import

			By("Create import DataObject containing the ConfigMap name")
			do = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-5", "import-do-name.yaml")))
			do.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Update Installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-5", "installation-pos.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to succeed")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(1))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
		})

		It("an installation should fail if an import value has the wrong type", func() {
			By("Create import DataObject containing the ConfigMap name")
			do1 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-6", "import-do-name-neg.yaml")))
			do1.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do1))

			By("Create import DataObject containing the ConfigMap namespace")
			do2 := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do2, path.Join(testdataDir, "installation-6", "import-do-namespace.yaml")))
			do2.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do2, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do2))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-6", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseFailed, 2*time.Minute))

			// Fix type of import value

			By("Update import DataObject containing the ConfigMap name")
			do1Old := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(do1), do1Old))
			do1 = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do1, path.Join(testdataDir, "installation-6", "import-do-name-pos.yaml")))
			do1.SetNamespace(state.Namespace)
			do1.ObjectMeta.ResourceVersion = do1Old.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, do1))

			By("Trigger a new reconciliation of the Installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-6", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to succeed")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(1))
			Expect(configMap.Data).To(HaveKeyWithValue("foo", "bar"))
		})
	})
}
