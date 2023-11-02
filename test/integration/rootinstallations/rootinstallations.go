// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package rootinstallations

import (
	"context"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func RootInstallationTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "rootinstallations")
	)

	Describe("Root Installations", func() {

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

		It("a root installation should trigger its successor", func() {
			var (
				do    = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				inst1 = &lsv1alpha1.Installation{}
				inst2 = &lsv1alpha1.Installation{}
			)

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, do, path.Join(testdataDir, "installation-root-trigger", "import-do-namespace.yaml")))

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create root installations")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst1, path.Join(testdataDir, "installation-root-trigger", "installation-1.yaml")))
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst2, path.Join(testdataDir, "installation-root-trigger", "installation-2.yaml")))

			By("Trigger 1st root installation")
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst1), inst1))
			instOld1 := inst1.DeepCopy()
			metav1.SetMetaDataAnnotation(&inst1.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
			utils.ExpectNoError(state.Client.Patch(ctx, inst1, client.MergeFrom(instOld1)))

			By("Wait for 1st installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst1, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Wait for 2nd installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst2, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMaps")
			expectedData := map[string]string{"foo": "bar", "key1": "value1"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example-x", expectedData))

			By("Check DataObjects with exports")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "do-name-3", "cm-example-x-x"))
			utils.ExpectNoError(utils.CheckDataObjectMap(ctx, state.State, "do-data-3", expectedData))

			By("Delete Installations")
			utils.ExpectNoError(f.Client.Delete(ctx, inst1))
			utils.ExpectNoError(f.Client.Delete(ctx, inst2))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst2, 2*time.Minute))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst1, 2*time.Minute))
		})
	})
}
