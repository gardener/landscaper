// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"path"
	"path/filepath"
	"time"

	k8sv1 "k8s.io/api/core/v1"

	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func TargetTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "targets")
	)

	Describe("Targets", func() {

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

		It("should export Targets", func() {

			// Create Installation that exports a Target "target-1"

			By("Create DataObjects with import data")
			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-exporter-1", "import-do-kubeconfig.yaml")))
			do.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-exporter-1", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed Target")
			targetKey := client.ObjectKey{Namespace: state.Namespace, Name: "target-1"}
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig := &lsv1alpha1.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig 1"))

			// Create Installation that exports a Target "target-2"

			By("Create DataObjects with import data")
			do = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-exporter-2", "import-do-kubeconfig.yaml")))
			do.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Installation")
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-exporter-2", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed Target")
			targetKey = client.ObjectKey{Namespace: state.Namespace, Name: "target-2"}
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig = &lsv1alpha1.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, &targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig 2"))

			// Create an Installation that imports a target list consisting of "target-1" and  "target-2"

			By("Create DataObjects with import data")
			do = &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-importer", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Installation")
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-importer", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("target-1", "dummy kubeconfig 1"))
			Expect(configMap.Data).To(HaveKeyWithValue("target-2", "dummy kubeconfig 2"))
		})

		It("should pass a Target to a subinstallation", func() {
			// A root installation imports a target (target-1.yaml) and passes it to its subinstallation.
			// The subinstallation writes name and kubeconfig of the target into a ConfigMap.

			By("Create DataObjects with import data")
			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-root-1", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Targets")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "installation-target-root-1", "target-1.yaml")))
			target.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(1))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 1"))

			// Update target

			By("Update Target")
			targetOld := &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(target), targetOld))
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "installation-target-root-1", "target-1-updated.yaml")))
			target.SetNamespace(state.Namespace)
			target.ObjectMeta.ResourceVersion = targetOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, target))

			By("Reconcile Installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(1))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 1 updated"))
		})
	})
}
