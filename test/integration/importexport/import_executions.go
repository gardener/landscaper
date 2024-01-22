// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"context"
	"path"
	"path/filepath"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ImportExecutionsTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "import-execution")
	)

	Describe("Import Executions", func() {

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

		It("should apply import executions", func() {
			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
			inst.SetNamespace(state.Namespace)
			inst.Spec.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
				"arbitraryImport": lsv1alpha1.NewAnyJSON([]byte(`"foo"`)),
				"mapToList": lsv1alpha1.NewAnyJSON([]byte(`{
"foo": "bar",
"a": "b"
}`)),
			}
			utils.ExpectNoError(state.Create(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployitem")
			dis, err := utils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(dis).To(HaveLen(1))
			di := dis[0]
			config := map[string]interface{}{}
			utils.ExpectNoError(yaml.Unmarshal(di.Spec.Configuration.Raw, &config))
			Expect(config).To(HaveKey("imps"))
			imports, ok := (config["imps"]).(map[string]interface{})
			Expect(ok).To(BeTrue(), "imps could not be converted into map")
			Expect(imports).To(HaveKeyWithValue("arbitraryImport", BeEquivalentTo("foo")))
			Expect(imports).To(HaveKeyWithValue("mapToList", ConsistOf(
				map[string]interface{}{
					"key":   "foo",
					"value": "bar",
				},
				map[string]interface{}{
					"key":   "a",
					"value": "b",
				},
			)))
		})

		It("should fail if import executions return errors", func() {
			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
			inst.SetNamespace(state.Namespace)
			inst.Spec.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
				"errors":          lsv1alpha1.NewAnyJSON([]byte(`["a foo error occurred"]`)),
				"arbitraryImport": lsv1alpha1.NewAnyJSON([]byte(`"foo"`)),
			}
			utils.ExpectNoError(state.Create(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Failed, 2*time.Minute))
			Expect(inst.Status.LastError.Message).To(ContainSubstring("a foo error occurred"))
		})

	})
}
