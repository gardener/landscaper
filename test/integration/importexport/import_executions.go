// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package importexport

import (
	"context"
	"path"
	"path/filepath"
	"time"

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

		BeforeEach(func() {
			ctx = context.Background()
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
				"passthrough":     lsv1alpha1.NewAnyJSON([]byte(`true`)),
				"arbitraryImport": lsv1alpha1.NewAnyJSON([]byte(`"foo"`)),
				"mapToList": lsv1alpha1.NewAnyJSON([]byte(`{
"foo": "bar",
"a": "b"
}`)),
			}
			utils.ExpectNoError(state.Create(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployitem")
			dis, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(dis).To(HaveLen(1))
			di := dis[0]
			config := map[string]interface{}{}
			utils.ExpectNoError(yaml.Unmarshal(di.Spec.Configuration.Raw, &config))
			Expect(config).To(HaveKey("imps"))
			imports, ok := (config["imps"]).(map[string]interface{})
			Expect(ok).To(BeTrue(), "imps could not be converted into map")
			Expect(imports).To(HaveKeyWithValue("passthrough", BeEquivalentTo(true)))
			Expect(imports).To(HaveKeyWithValue("arbitraryImport", BeEquivalentTo("foo")))
			Expect(imports).To(HaveKeyWithValue("mapToList", BeEquivalentTo([]map[string]string{
				{
					"key":   "foo",
					"value": "bar",
				},
				{
					"key":   "a",
					"value": "b",
				},
			})))
		})

		It("should only pass through imports if explicitly configured in the template", func() {
			By("Create installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.ReadResourceFromFile(inst, path.Join(testdataDir, "installation-1", "installation.yaml")))
			inst.SetNamespace(state.Namespace)
			inst.Spec.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
				"passthrough":     lsv1alpha1.NewAnyJSON([]byte(`false`)),
				"arbitraryImport": lsv1alpha1.NewAnyJSON([]byte(`"foo"`)),
				"mapToList": lsv1alpha1.NewAnyJSON([]byte(`{
"foo": "bar",
"a": "b"
}`)),
			}
			utils.ExpectNoError(state.Create(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployitem")
			dis, err := lsutils.GetDeployItemsOfInstallation(ctx, f.Client, inst)
			utils.ExpectNoError(err)
			Expect(dis).To(HaveLen(1))
			di := dis[0]
			config := map[string]interface{}{}
			utils.ExpectNoError(yaml.Unmarshal(di.Spec.Configuration.Raw, &config))
			Expect(config).To(HaveKey("imps"))
			imports, ok := (config["imps"]).(map[string]interface{})
			Expect(ok).To(BeTrue(), "imps could not be converted into map")
			Expect(imports).ToNot(HaveKey("passthrough"))
			Expect(imports).ToNot(HaveKey("arbitraryImport"))
			Expect(imports).To(HaveKeyWithValue("mapToList", BeEquivalentTo([]map[string]string{
				{
					"key":   "foo",
					"value": "bar",
				},
				{
					"key":   "a",
					"value": "b",
				},
			})))
		})

	})
}
