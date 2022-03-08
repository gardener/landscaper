// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/landscaper/pkg/utils"
)

var _ = Describe("Cyclic Dependency Determination Tests", func() {

	Context("DetermineCyclicDependencyDetails", func() {

		It("should not detect cyclic dependencies if there aren't any", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString(),
			}

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, nil)
			Expect(cycles).To(HaveLen(0))
		})

		It("should detect simple cyclic dependencies", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("c"),
			}

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, nil)
			Expect(cycles).To(HaveLen(1))
			cmp := utils.NewDependencyCycle("a")
			cmp.Add("c", nil)
			cmp.Add("b", nil)
			cmp.Close(nil)
			Expect(cycles[0]).To(beEqualToCycle(cmp))
		})

		It("should detect one-elemented cyclic dependencies", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("a"),
			}

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, nil)
			Expect(cycles).To(HaveLen(1))
			Expect(cycles[0]).To(beEqualToCycle(utils.NewDependencyCycle("a").Close(nil)))
		})

		It("should detect multiple independent cycles", func() {
			deps := map[string]sets.String{
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("b"),
			}

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, nil)
			cmp1 := utils.NewDependencyCycle("a")
			cmp1.Add("b", nil)
			cmp1.Close(nil)
			cmp2 := utils.NewDependencyCycle("c")
			cmp2.Add("d", nil)
			cmp2.Close(nil)
			Expect(cycles).To(ConsistOf(beEqualToCycle(cmp1), beEqualToCycle(cmp2)))
		})

		It("should detect multiple connected cycles", func() {
			deps := map[string]sets.String{
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d", "b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("d"),
			}

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, nil)
			cmp1 := utils.NewDependencyCycle("a")
			cmp1.Add("d", nil)
			cmp1.Add("c", nil)
			cmp1.Add("b", nil)
			cmp1.Close(nil)
			cmp2 := utils.NewDependencyCycle("c")
			cmp2.Add("d", nil)
			cmp2.Close(nil)
			Expect(cycles).To(ConsistOf(beEqualToCycle(cmp1), beEqualToCycle(cmp2)))
		})

		It("should detect simple cyclic dependencies with import relationships", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("c"),
			}
			impRels := utils.ImportRelationships{}
			impRels.Add("c", "a", "c_to_a").Add("a", "b", "a_to_b").Add("b", "c", "b_to_c")

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, impRels)
			Expect(cycles).To(HaveLen(1))
			cmp := utils.NewDependencyCycle("a")
			tmp, _ := impRels.Get("c", "a")
			cmp.Add("c", tmp)
			tmp, _ = impRels.Get("b", "c")
			cmp.Add("b", tmp)
			tmp, _ = impRels.Get("a", "b")
			cmp.Close(tmp)
			Expect(cycles[0]).To(beEqualToCycle(cmp))
		})

		It("should detect one-elemented cyclic dependencies with import relationships", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("a"),
			}
			impRels := utils.ImportRelationships{}
			impRels.Add("a", "a", "a_to_a")

			cycles := utils.DetermineCyclicDependencyDetails(sets.StringKeySet(deps), deps, impRels)
			Expect(cycles).To(HaveLen(1))
			tmp, _ := impRels.Get("a", "a")
			Expect(cycles[0]).To(beEqualToCycle(utils.NewDependencyCycle("a").Close(tmp)))
		})

	})

	Context("DependencyCycle", func() {

		Context("String", func() {

			It("should print the cycle", func() {
				c := utils.NewDependencyCycle("a")
				c.Add("b", nil)
				c.Add("c", nil)
				Expect(c.String()).To(Equal("a -> b -> c"))
			})

			It("should print the cycle and add the starting element if it is closed", func() {
				c := utils.NewDependencyCycle("a")
				c.Add("b", nil)
				c.Add("c", nil)
				c.Close(nil)
				Expect(c.String()).To(Equal("a -> b -> c -> a"))
			})

			It("should print the imports if given", func() {
				c := utils.NewDependencyCycle("a")
				c.Add("b", sets.NewString().Insert("foo"))
				c.Add("c", sets.NewString().Insert("bar"))
				c.Close(sets.NewString().Insert("foobar", "baz"))
				Expect(c.String()).To(Equal("a -[foo]-> b -[bar]-> c -[baz, foobar]-> a"))
			})

		})

	})

})

func beEqualToCycle(cmp *utils.DependencyCycle) types.GomegaMatcher {
	return And(
		Not(BeNil()),
		WithTransform(func(dc *utils.DependencyCycle) bool {
			return cmp != nil && dc.Equal(cmp)
		}, BeTrue()),
	)
}
