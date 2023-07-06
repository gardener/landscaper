// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targets

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
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
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
			var (
				do1   = &lsv1alpha1.DataObject{} // contains a dummy kubeconfig for a Target to be deployed by the Installation
				inst1 = &lsv1alpha1.Installation{}
			)

			// Create Installation that exports a Target "target-1"

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-target-exporter-1", "import-do-kubeconfig.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst1, path.Join(testdataDir, "installation-target-exporter-1", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst1, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed Target")
			targetKey := client.ObjectKey{Namespace: state.Namespace, Name: "target-1"}
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig := &targettypes.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig 1"))

			// Create Installation that exports a Target "target-2"

			var (
				do2   = &lsv1alpha1.DataObject{} // contains a dummy kubeconfig for a Target to be deployed by the Installation
				inst2 = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do2, path.Join(testdataDir, "installation-target-exporter-2", "import-do-kubeconfig.yaml")))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst2, path.Join(testdataDir, "installation-target-exporter-2", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst2, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed Target")
			targetKey = client.ObjectKey{Namespace: state.Namespace, Name: "target-2"}
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig = &targettypes.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, &targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig 2"))

			// Create an Installation that imports a target list consisting of "target-1" and  "target-2"

			var (
				do3   = &lsv1alpha1.DataObject{}
				inst3 = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do3, path.Join(testdataDir, "installation-target-importer-1", "import-do-namespace.yaml")))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst3, path.Join(testdataDir, "installation-target-importer-1", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst3, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"target-1": "dummy kubeconfig 1", "target-2": "dummy kubeconfig 2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-kubeconfigs", expectedData))
		})

		It("should create a DeployItem for each Target in a TargetList", func() {
			var (
				do   = &lsv1alpha1.DataObject{}
				inst = &lsv1alpha1.Installation{}
			)

			// Create Installation that imports a TargetList with two Targets "target-1" and "target-2", and generates
			// a DeployItem for each of them.

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do, path.Join(testdataDir, "installation-target-importer-2", "import-do-namespace.yaml")))

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
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-target-importer-2", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData := map[string]string{"targetName": "target-1", "kubeconfig": "dummy kubeconfig 1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-kubeconfig-0", expectedData))

			expectedData = map[string]string{"targetName": "target-2", "kubeconfig": "dummy kubeconfig 2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-kubeconfig-1", expectedData))
		})

		It("should pass a Target and TargetList to a subinstallation", func() {
			// A root installation imports a target (target-1.yaml) and passes it to its subinstallation.
			// The subinstallation writes name and kubeconfig of the target into a ConfigMap.
			// Next, we update the Target and check that the deployed ConfigMap is changed accordingly.
			// Then the installation is updated, so that it imports a TargetList and passes it to a subinstallation.
			// Finally one of the Targets of the TargetList is updated.
			// The root installation templates its subinstallation, i.e. it uses a SubinstallationExecution; this is
			// necessary because the Target and TargetList that are passed to the subinstallation are optional parameters.

			var (
				do   = &lsv1alpha1.DataObject{}
				inst = &lsv1alpha1.Installation{}
			)

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do, path.Join(testdataDir, "installation-target-root-1", "import-do-namespace.yaml")))

			By("Create Targets")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(utils.ReadResourceFromFile(target, path.Join(testdataDir, "installation-target-root-1", "target-1.yaml")))
			target.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-target-root-1", "installation-1.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

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
			utils.ExpectNoError(state.Update(ctx, target))

			By("Reconcile Installation")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-target-root-1", "installation-1.yaml")))
			inst.Namespace = state.Namespace
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(state.Update(ctx, inst))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

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
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-target-root-1", "installation-2.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

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
			utils.ExpectNoError(state.Update(ctx, target2))

			By("Reconcile Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-target-root-1", "installation-2.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed configmap")
			configMapKey = client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap = &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(2))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 1 updated"))
			Expect(configMap.Data).To(ContainElement("dummy kubeconfig 2 updated"))
		})

		It("should use a Target that references a Secret", func() {
			var (
				do   = &lsv1alpha1.DataObject{}
				inst = &lsv1alpha1.Installation{}
			)

			// Create an Installation that uses a Target with a reference to a Secret.

			By("Create DataObject with import data")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do, path.Join(testdataDir, "installation-target-secretref", "import-do-namespace.yaml")))

			By("Create Target and Secret")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			target, secret, err := utils.BuildTargetAndSecretFromKubernetesTarget(target)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, secret))
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-target-secretref", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-kubeconfigs"}
			configMap := &k8sv1.ConfigMap{}
			utils.ExpectNoError(f.Client.Get(ctx, configMapKey, configMap))
			Expect(configMap.Data).To(HaveLen(0))
		})

		It("should update an exported Target", func() {
			var (
				do1   = &lsv1alpha1.DataObject{} // contains a dummy kubeconfig for a Target to be deployed by the Installation
				inst1 = &lsv1alpha1.Installation{}
			)

			// Create root Installation with subinstallation. The root Installation exports the Target that was exported
			// by the subinstallation.

			By("Create DataObjects with import data")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-target-exporter-root", "import-do-kubeconfig-1.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installation")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst1, path.Join(testdataDir, "installation-target-exporter-root", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst1, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check exported Target")
			targetKey := client.ObjectKey{Namespace: state.Namespace, Name: "target-1"}
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig := &targettypes.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig"))

			// Update DataObject

			By("Update DataObjects with import data")
			utils.ExpectNoError(utils.UpdateDataObjectFromFile(ctx, state.State, do1, path.Join(testdataDir, "installation-target-exporter-root", "import-do-kubeconfig-2.yaml")))

			By("Reconcile Installation")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst1, path.Join(testdataDir, "installation-target-exporter-root", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst1, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check exported Target")
			target = &lsv1alpha1.Target{}
			utils.ExpectNoError(f.Client.Get(ctx, targetKey, target))
			targetConfig = &targettypes.KubernetesClusterTargetConfig{}
			utils.GetTargetConfiguration(target, targetConfig)
			Expect(*targetConfig.Kubeconfig.StrVal).To(Equal("dummy kubeconfig modified"))
		})
	})
}
