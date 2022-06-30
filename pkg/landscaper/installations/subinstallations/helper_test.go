// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations_test

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/sets"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
	"github.com/gardener/landscaper/pkg/utils"
)

var _ = Describe("SubinstallationsHelper", func() {

	Context("DependencyComputation", func() {

		It("should not compute dependencies for independent installation templates", func() {
			deps := map[string][]string{
				"a": nil,
				"b": nil,
				"c": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			computedDeps, impRels := subinstallations.ComputeInstallationDependencies(subinstallations.AbstractInstallationTemplates(tmpls))
			for k, v := range computedDeps {
				Expect(v).To(BeEmpty(), "entry %q has non-empty dependency list", k)
			}
			Expect(impRels).To(BeEmpty())
		})

		It("should correctly detect data dependencies", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			computedDeps, impRels := subinstallations.ComputeInstallationDependencies(subinstallations.AbstractInstallationTemplates(tmpls))
			Expect(computedDeps).To(HaveKeyWithValue("b", HaveKey("a")))
			Expect(impRels).To(HaveLen(1))
			Expect(impRels).To(HaveKeyWithValue(utils.RelationshipTuple{Exporting: "a", Importing: "b"}, sets.NewString().Insert("a_data")))
		})
		It("should correctly detect target dependencies", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(targetDependency))
			computedDeps, impRels := subinstallations.ComputeInstallationDependencies(subinstallations.AbstractInstallationTemplates(tmpls))
			Expect(computedDeps).To(HaveKeyWithValue("b", HaveKey("a")))
			Expect(impRels).To(HaveLen(1))
			Expect(impRels).To(HaveKeyWithValue(utils.RelationshipTuple{Exporting: "a", Importing: "b"}, sets.NewString().Insert("a_target")))
		})
		It("should correctly detect dependencies defined in exportDataMappings", func() {
			deps := map[string][]string{
				"a": nil,
				"b": {"a"},
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(mappingDependency))
			computedDeps, impRels := subinstallations.ComputeInstallationDependencies(subinstallations.AbstractInstallationTemplates(tmpls))
			Expect(computedDeps).To(HaveKeyWithValue("b", HaveKey("a")))
			Expect(impRels).To(HaveLen(1))
			Expect(impRels).To(HaveKeyWithValue(utils.RelationshipTuple{Exporting: "a", Importing: "b"}, sets.NewString().Insert("a_mapping")))
		})

	})

	Context("InstallationTemplateOrdering", func() {

		It("should return independent installation templates in the same order they were given", func() {
			deps := map[string][]string{
				"a": nil,
				"b": nil,
				"c": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			ordered, err := subinstallations.OrderInstallationTemplates(tmpls)
			Expect(err).ToNot(HaveOccurred())
			Expect(ordered).To(Equal(tmpls))
		})

		It("should correctly order based on data dependencies", func() {
			deps := map[string][]string{
				"a": {"b"},
				"b": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(dataDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			ordered, err := subinstallations.OrderInstallationTemplates(tmpls)
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
			ordered, err := subinstallations.OrderInstallationTemplates(tmpls)
			Expect(err).ToNot(HaveOccurred())
			Expect(installationTemplatesToNames(ordered)).To(Equal([]string{"b", "a"}))
		})
		It("should correctly order based on dependencies defined in exportDataMappings", func() {
			deps := map[string][]string{
				"a": {"b"},
				"b": nil,
			}
			tmpls := generateSubinstallationTemplates(deps, newDependencyProvider(mappingDependency))
			sortInstallationTemplatesAlphabetically(tmpls)
			ordered, err := subinstallations.OrderInstallationTemplates(tmpls)
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
			ordered, err := subinstallations.OrderInstallationTemplates(tmpls)
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
			_, err := subinstallations.OrderInstallationTemplates(tmpls)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Op: EnsureNestedInstallations - Reason: OrderNestedInstallationTemplates - Message: The following cyclic dependencies have been found in the nested installation templates: {c -[e_data]-> e -[d_mapping]-> d -[c_target]-> c}"))
		})

	})

})

type dependencyMode string

const (
	dataDependency    = dependencyMode("data")
	targetDependency  = dependencyMode("target")
	mappingDependency = dependencyMode("mapping")
	mixedDependency   = dependencyMode("mixed")
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
		cur := d.count % 3
		switch cur {
		case 0:
			mode = dataDependency
		case 1:
			mode = targetDependency
		case 2:
			mode = mappingDependency
		}
		d.count++
	}
	return string(mode)
}

// generateSubinstallationTemplates describes dependencies between the subinstallation templates which should be created
// There is expected to be one key for each installation template
// and the value should list the names of all other installation templates this one depends on.
// The dependency provider returns one of 'data', 'target', or 'mapping' via its 'nextMode' function, which controls whether the import is satisfied by a
// data export, target export, or something exported in the exportDataMappings, respectively.
func generateSubinstallationTemplates(deps map[string][]string, dProv dependencyProvider) []*lsv1alpha1.InstallationTemplate {
	res := []*lsv1alpha1.InstallationTemplate{}
	for k, v := range deps {
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
			ExportDataMappings: map[string]lsv1alpha1.AnyJSON{
				fmt.Sprintf("%s_mapping", k): lsv1alpha1.NewAnyJSON([]byte("foo_mapping_value")),
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
