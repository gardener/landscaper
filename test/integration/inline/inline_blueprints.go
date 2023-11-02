// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"context"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
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

func InlineBlueprintTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "inline")
	)

	Describe("Inline Blueprints", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %w", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("an installation should use an inline blueprint", func() {
			var (
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				doEntry     = &lsv1alpha1.DataObject{} // contains an entry of the ConfigMap
				inst        = &lsv1alpha1.Installation{}
			)

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-inline-blueprint", "import-do-namespace.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installations with inline blueprint")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-blueprint", "installation-1.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-1", expectedData))

			By("Update Installation: add DeployItems")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-blueprint", "installation-2.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-1", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-2", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-3", expectedData))

			// Update Installation: one DeployItem updated, two DeployItems deleted, one deployed ConfigMap, new import

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateDataObjectFromFile(ctx, state.State, doEntry, path.Join(testdataDir, "installation-inline-blueprint", "import-do-entry.yaml")))

			By("Update Installation: add imports and exports, remove DeployItems")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-blueprint", "installation-3.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData = map[string]string{"key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-1", expectedData))

			for _, cmName := range []string{"cm-2", "cm-3"} {
				configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: cmName}
				configMap := &k8sv1.ConfigMap{}
				err := f.Client.Get(ctx, configMapKey, configMap)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}

			By("Check DataObjects with exports")
			expectedData = map[string]string{"key": "key2", "value": "value2"}
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "do-entry-out", expectedData))

			// Update Installation: remove imports

			By("Update Installation: remove imports")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-inline-blueprint", "installation-4.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData = map[string]string{"foo": "bar"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-1", expectedData))

			// Delete Installation

			By("Delete Installations")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute))

			By("Check deletion of ConfigMaps")
			for _, cmName := range []string{"cm-1", "cm-2", "cm-3"} {
				configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: cmName}
				configMap := &k8sv1.ConfigMap{}
				err := f.Client.Get(ctx, configMapKey, configMap)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}
		})
	})
}
