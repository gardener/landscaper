// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"fmt"
	"sort"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Cyclic Dependency Determination Tests", func() {

	Context("OrderTemplates", func() {

		It("should return independent installation templates in the same order they were given", func() {
			deps := map[string][]string{
				"a": nil,
				"b": nil,
				"c": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			_, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should correctly detect data dependencies", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			orderedTmpls, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(orderedTmpls[0].Name == "a").To(BeTrue())
			Expect(orderedTmpls[1].Name == "b").To(BeTrue())
		})

		It("should correctly order based on data dependencies", func() {
			deps := map[string][]string{
				"a": {"b"},
				"b": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			ordered, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(installationTemplatesToNames(ordered)).To(Equal([]string{"b", "a"}))
		})

		It("should correctly order based on target dependencies", func() {
			deps := map[string][]string{
				"a": {"b"},
				"b": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(targetDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			ordered, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(installationTemplatesToNames(ordered)).To(Equal([]string{"b", "a"}))
		})

		It("should correctly order based on dependencies", func() {
			deps := map[string][]string{
				"a": {"f"},
				"b": nil,
				"c": {"b"},
				"d": {"b"},
				"e": {"b", "c"},
				"f": {"e"},
				"g": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(mixedDependency))
			sortInstallationTemplatesAlphabetically(tmpls)

			ordered, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).ToNot(HaveOccurred())
			indices := stringSliceToIndexMap(installationTemplatesToNames(ordered))
			Expect(indices["b"]).To(BeNumerically("<", indices["c"]))
			Expect(indices["b"]).To(BeNumerically("<", indices["d"]))
			Expect(indices["b"]).To(BeNumerically("<", indices["e"]))
			Expect(indices["b"]).To(BeNumerically("<", indices["f"]))
			Expect(indices["b"]).To(BeNumerically("<", indices["a"]))
			Expect(indices["c"]).To(BeNumerically("<", indices["e"]))
			Expect(indices["c"]).To(BeNumerically("<", indices["f"]))
			Expect(indices["c"]).To(BeNumerically("<", indices["a"]))
			Expect(indices["d"]).To(BeNumerically("<", indices["a"]))
			Expect(indices["e"]).To(BeNumerically("<", indices["f"]))
			Expect(indices["e"]).To(BeNumerically("<", indices["a"]))
			Expect(indices["f"]).To(BeNumerically("<", indices["a"]))
		})

		It("should detect cycles", func() {
			deps := map[string][]string{
				"a": nil,
				"b": nil,
				"c": {"e"},
				"d": {"c"},
				"e": {"d"},
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(mixedDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			_, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(SatisfyAll(ContainSubstring("c -{depends_on}-> e"), ContainSubstring("d -{depends_on}-> c"), ContainSubstring("e -{depends_on}-> d")))
		})

		It("should detect duplicate data exports", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
				"c": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			tmpls[0].Exports.Data = append(tmpls[0].Exports.Data, lsv1alpha1.DataExport{Name: "addExp", DataRef: "addExp"})
			tmpls[len(tmpls)-1].Exports.Data = tmpls[0].Exports.Data
			_, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).To(HaveOccurred())
			matchers := []types.GomegaMatcher{}
			for _, exp := range tmpls[0].Exports.Data {
				matchers = append(matchers, ContainSubstring("'%s' is exported by [%s]", exp.DataRef, strings.Join([]string{tmpls[0].Name, tmpls[len(tmpls)-1].Name}, ", ")))
			}
			Expect(err.Error()).To(SatisfyAll(matchers...))
		})

		It("should detect duplicate target exports", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
				"c": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(targetDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			tmpls[0].Exports.Targets = append(tmpls[0].Exports.Targets, lsv1alpha1.TargetExport{Name: "addExp", Target: "addExp"})
			tmpls[len(tmpls)-1].Exports.Targets = tmpls[0].Exports.Targets
			_, err := CheckForCyclesAndDuplicateExports(tmpls, true)
			Expect(err).To(HaveOccurred())
			matchers := []types.GomegaMatcher{}
			for _, exp := range tmpls[0].Exports.Targets {
				matchers = append(matchers, ContainSubstring("'%s' is exported by [%s]", exp.Target, strings.Join([]string{tmpls[0].Name, tmpls[len(tmpls)-1].Name}, ", ")))
			}
			Expect(err.Error()).To(SatisfyAll(matchers...))
		})

	})
})

type dependencyMode string

const (
	dataDependency   = dependencyMode("data")
	targetDependency = dependencyMode("target")
	mixedDependency  = dependencyMode("mixed")
)

// dependencyProvider is a helper struct to iterate over dependency modes
type dependencyProvider struct {
	count int
	mode  dependencyMode
}

func newDependencyProvider(mode dependencyMode) dependencyProvider {
	return dependencyProvider{
		mode: mode,
	}
}

func (d *dependencyProvider) nextMode() string {
	mode := d.mode
	if mode == mixedDependency {
		cur := d.count % 2
		switch cur {
		case 0:
			mode = dataDependency
		case 1:
			mode = targetDependency
		}
		d.count = (d.count + 1) % 2
	}
	return string(mode)
}

// generateSubinstallationTemplates describes dependencies between the subinstallation templates which should be created
// There is expected to be one key for each installation template
// and the value should list the names of all other installation templates this one depends on.
// The dependency provider returns one of 'data' or 'target', via its 'nextMode' function, which controls whether the import is satisfied by a
// data export or target export, respectively.
func generateSubinstallationTemplates(deps map[string][]string, dProv dependencyProvider) []*lsv1alpha1.InstallationTemplate {
	res := []*lsv1alpha1.InstallationTemplate{}
	// sort map keys to make iteration order deterministic
	keys := []string{}
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := deps[k]
		tmpl := &lsv1alpha1.InstallationTemplate{
			Name: k,
			Imports: lsv1alpha1.InstallationImports{
				Data: []lsv1alpha1.DataImport{},
			},
			Exports: lsv1alpha1.InstallationExports{
				Data: []lsv1alpha1.DataExport{
					{
						Name:    "foo_data",
						DataRef: fmt.Sprintf("%s_data", k),
					},
				},
				Targets: []lsv1alpha1.TargetExport{
					{
						Name:   "foo_target",
						Target: fmt.Sprintf("%s_target", k),
					},
				},
			},
		}
		for i, dep := range v {
			mode := dProv.nextMode()
			if mode == string(targetDependency) {
				tmpl.Imports.Targets = append(tmpl.Imports.Targets, lsv1alpha1.TargetImport{
					Name:   fmt.Sprintf("%s_%d", k, i),
					Target: fmt.Sprintf("%s_%s", dep, mode),
				})
			} else {
				tmpl.Imports.Data = append(tmpl.Imports.Data, lsv1alpha1.DataImport{
					Name:    fmt.Sprintf("%s_%d", k, i),
					DataRef: fmt.Sprintf("%s_%s", dep, mode),
				})
			}
		}
		res = append(res, tmpl)
	}
	return res
}

// sortInstallationTemplatesAlphabetically sorts a slice of installation templates by name
// This can be used to get templates into a deterministic order before checking for ordering by dependencies.
func sortInstallationTemplatesAlphabetically(templates []*lsv1alpha1.InstallationTemplate) {
	sort.Slice(templates, func(i, j int) bool {
		return strings.Compare(templates[i].Name, templates[j].Name) < 0
	})
}

// installationTemplatesToNames takes a slice of installation templates and returns a slice of their names, in the same order
func installationTemplatesToNames(templates []*lsv1alpha1.InstallationTemplate) []string {
	res := make([]string, len(templates))
	for i, tmpl := range templates {
		res[i] = tmpl.Name
	}
	return res
}

// stringSliceToIndexMap takes a slice of unique(!) strings and returns a mapping of these strings to their respective indices in the slice
func stringSliceToIndexMap(data []string) map[string]int {
	res := map[string]int{}
	for i, elem := range data {
		res[elem] = i
	}
	return res
}
