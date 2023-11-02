// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"context"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
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

		It("should process subinstallations in the correct order", func() {
			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with chain of four subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-chain", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check processing order of subinstallations")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-track", "ABCD"))

			// Update installation

			By("Update Installation with changed dependencies between subinstallations")
			utils.ExpectNoError(utils.UpdateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-join", "installation.yaml")))

			By("Wait for installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check changed processing order of subinstallations")
			utils.ExpectNoError(utils.CheckDataObjectString(ctx, state.State, "export-do-track", "(A|B|C)D"))
		})

		It("should detect a dependency cycle", func() {
			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with cycle of subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-cycle", "installation.yaml")))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))
		})

		It("should detect an export conflict", func() {
			By("Create target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig, true)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create installation with an export conflict between subinstallations")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-conflict", "installation.yaml")))

			By("Wait for installation to fail")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))
		})
	})
}
