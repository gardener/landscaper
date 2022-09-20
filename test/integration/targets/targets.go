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
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-importer-1", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Installation")
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-importer-1", "installation.yaml")))
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

		It("should create a DeployItem for each Target in a TargetList", func() {

			// Create Installation that imports a TargetList with two Targets "target-1" and "target-2", and generates
			// a DeployItem for each of them.

			By("Create DataObjects with import data")
			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-importer-2", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Targets")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "installation-target-importer-2", "target-1.yaml")))
			target.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target))

			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "installation-target-importer-2", "target-2.yaml")))
			target.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-importer-2", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfig-0"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("targetName", "target-1"))
			Expect(configMap.Data).To(HaveKeyWithValue("kubeconfig", "dummy kubeconfig 1"))

			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfig-1"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(HaveKeyWithValue("targetName", "target-2"))
			Expect(configMap.Data).To(HaveKeyWithValue("kubeconfig", "dummy kubeconfig 2"))
		})

		It("should pass a Target and TargetList to a subinstallation", func() {
			// A root installation imports a target (target-1.yaml) and passes it to its subinstallation.
			// The subinstallation writes name and kubeconfig of the target into a ConfigMap.
			// Next, we update the Target and check that the deployed ConfigMap is changed accordingly.
			// Then the installation is updated, so that it imports a TargetList and passes it to a subinstallation.
			// Finally one of the Targets of the TargetList is updated.
			// The root installation templates its subinstallation, i.e. it uses a SubinstallationExecution; this is
			// necessary because the Target and TargetList that are passed to the subinstallation are optional parameters.

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
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation-1.yaml")))
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
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation-1.yaml")))
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

			// Update Installation so that it imports a TargetList and passes it to a subinstallation

			By("Create Target")
			target2 := &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target2, path.Join(testdataDir, "installation-target-root-1", "target-2.yaml")))
			target2.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target2))

			By("Update Installation so that it imports a TargetList")
			instOld = &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation-2.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 1 updated"))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 2"))

			// Update the 2nd Target of the TargetList

			By("Update one Target of the TargetList")
			target2Old := &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(target2), target2Old))
			target2 = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target2, path.Join(testdataDir, "installation-target-root-1", "target-2-updated.yaml")))
			target2.SetNamespace(state.Namespace)
			target2.ObjectMeta.ResourceVersion = target2Old.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, target2))

			By("Reconcile Installation")
			instOld = &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation-2.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 1 updated"))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 2 updated"))
		})

		It("should use a Target that references a Secret", func() {

			// Create an Installation that uses a Target with a reference to a Secret.

			By("Create DataObject with import data")
			do := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.ReadResourceFromFile(do, path.Join(testdataDir, "installation-target-secretref", "import-do-namespace.yaml")))
			do.SetNamespace(state.Namespace)
			utils.SetDataObjectData(do, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, do))

			By("Create Target and Secret")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			target, secret, err := utils.BuildTargetAndSecretFromKubernetesTarget(target)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, secret))
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-secretref", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(0))
		})
	})
}
