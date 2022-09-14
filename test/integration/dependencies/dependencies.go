package dependencies

import (
	"context"
	"path"
	"path/filepath"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func DependencyTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "dependencies")
	)

	Describe("Dependencies", func() {

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

		It("should process subinstallations in the correct order", func() {

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with chain of four subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-chain", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check DataObject with export")
			exportDo := &lsv1alpha1.DataObject{}
			exportDoKey := client.ObjectKey{Name: "export-do-track", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			track := ""
			utils.GetDataObjectData(exportDo, &track)
			Expect(track).To(Equal("ABCD"))

			// Update installation

			By("Update installation with changed dependencies between subinstallations")
			instOld := &lsv1alpha1.Installation{}
			utils.ExpectNoError(f.Client.Get(ctx, client.ObjectKeyFromObject(inst), instOld))
			inst = &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-join", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			inst.ObjectMeta.ResourceVersion = instOld.ObjectMeta.ResourceVersion
			utils.ExpectNoError(f.Client.Update(ctx, inst))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseSucceeded, 2*time.Minute))

			By("Check DataObjects with exports")
			exportDo = &lsv1alpha1.DataObject{}
			exportDoKey = client.ObjectKey{Name: "export-do-track", Namespace: state.Namespace}
			utils.ExpectNoError(f.Client.Get(ctx, exportDoKey, exportDo))
			track = ""
			utils.GetDataObjectData(exportDo, &track)
			Expect(track).To(Equal("(A|B|C)D"))
		})

		It("should detect a dependency cycle", func() {

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with cycle of subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-cycle", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseFailed, 2*time.Minute))
		})

		It("should detect an export conflict", func() {

			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with an export conflict between subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-conflict", "installation.yaml")))
			utils.SetInstallationNamespace(inst, state.Namespace)
			utils.ExpectNoError(state.Create(ctx, inst))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhaseFailed, 2*time.Minute))
		})
	})
}
