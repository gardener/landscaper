// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/sets"
)

var _ = Describe("Cyclic Dependency Determination Tests", func() {

	Context("DetermineCyclicDependencies", func() {

		It("should not detect cyclic dependencies in one element cycle", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("a"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeTrue())
		})

		It("should not detect cyclic dependencies if there aren't any", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString(),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeFalse())
		})

		It("should detect simple cyclic dependencies", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("c"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeTrue())
		})

		It("should detect one-elemented cyclic dependencies", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("a"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeTrue())
		})

		It("should detect multiple independent cycles", func() {
			deps := map[string]sets.String{
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("b"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeTrue())
		})

		It("should detect multiple connected cycles", func() {
			deps := map[string]sets.String{
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("d"),
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d", "b"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			fmt.Println(cycle)
			Expect(hasCycle).To(BeTrue())
		})

		It("should detect multiple connected cycles", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("c"),
				"b": sets.NewString().Insert("f", "e"),
				"c": sets.NewString().Insert("d", "e"),
				"d": sets.NewString().Insert("i"),
				"e": sets.NewString().Insert("h"),
				"f": sets.NewString().Insert("e", "g"),
				"g": sets.NewString().Insert("h"),
				"h": sets.NewString().Insert("i", "j"),
				"i": sets.NewString(),
				"j": sets.NewString(),
			}

			g := newGraph(deps)
			reverseOrder, err := g.getReverseOrder()
			Expect(err).ToNot(HaveOccurred())
			indices := stringSliceToIndexMap(reverseOrder)
			Expect(indices["j"]).To(BeNumerically("<", indices["h"]))
			Expect(indices["i"]).To(BeNumerically("<", indices["h"]))
			Expect(indices["i"]).To(BeNumerically("<", indices["d"]))
			Expect(indices["h"]).To(BeNumerically("<", indices["g"]))
			Expect(indices["h"]).To(BeNumerically("<", indices["e"]))
			Expect(indices["g"]).To(BeNumerically("<", indices["f"]))
			Expect(indices["f"]).To(BeNumerically("<", indices["b"]))
			Expect(indices["e"]).To(BeNumerically("<", indices["f"]))
			Expect(indices["e"]).To(BeNumerically("<", indices["b"]))
			Expect(indices["d"]).To(BeNumerically("<", indices["c"]))
			Expect(indices["c"]).To(BeNumerically("<", indices["a"]))
		})
	})
})
