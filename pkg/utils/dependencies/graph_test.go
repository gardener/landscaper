// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	gomegamatchers "github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

var _ = Describe("Cyclic Dependency Determination Tests", func() {

	Context("DetermineCyclicDependencies", func() {

		It("should not detect cyclic dependencies if there aren't any", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString(),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			Expect(hasCycle).To(BeFalse())
			Expect(cycle).To(BeEmpty())
		})

		It("should detect simple cyclic dependencies", func() {
			deps := map[string]sets.String{
				"c": sets.NewString().Insert("b"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("c"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			Expect(hasCycle).To(BeTrue())
			Expect(cycle).To(CyclicEqual("a", "b", "c"))
		})

		It("should detect one-elemented cyclic dependencies", func() {
			deps := map[string]sets.String{
				"a": sets.NewString().Insert("a"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			Expect(hasCycle).To(BeTrue())
			Expect(cycle).To(CyclicEqual("a"))
		})

		It("should detect multiple independent cycles", func() {
			deps := map[string]sets.String{
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d"),
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("b"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			Expect(hasCycle).To(BeTrue())
			Expect(cycle).To(SatisfyAny(CyclicEqual("c", "d"), CyclicEqual("a", "b")))
		})

		It("should detect multiple connected cycles", func() {
			deps := map[string]sets.String{
				"b": sets.NewString().Insert("a"),
				"a": sets.NewString().Insert("d"),
				"d": sets.NewString().Insert("c"),
				"c": sets.NewString().Insert("d", "b"),
			}

			hasCycle, cycle := newGraph(deps).hasCycle()
			Expect(hasCycle).To(BeTrue())
			Expect(cycle).To(SatisfyAny(CyclicEqual("c", "d", "a", "b"), CyclicEqual("c", "d")))
		})

		It("should order the graph correctly according to its edges", func() {
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

type CyclicEqualMatcher struct {
	Elements                            []interface{}
	lengthMismatch                      bool
	firstExpectedNotFound               bool
	first, actualSecond, expectedSecond interface{}
}

func (ce CyclicEqualMatcher) String() string {
	return fmt.Sprintf("{cyclically equal to %s}", presentable(ce.Elements))
}

// CyclicEquals succeeds if both arrays/slices contain the same elements in the same order.
// The arrays/slices are considered to not have a beginning or end, but be wrapped around,
// so [a, b, c] and [c, a, b] would be considered equal, while [a, c, b] is different from both.
// If the first and the last element of either actual or expected are equal, the last element is removed from the
// respective array/slice before comparison.
// The Equal matcher will be used for comparison.
func CyclicEqual(elements ...interface{}) types.GomegaMatcher {
	return &CyclicEqualMatcher{
		Elements:              elements,
		lengthMismatch:        false,
		firstExpectedNotFound: false,
	}
}

func (ce *CyclicEqualMatcher) Match(actual interface{}) (success bool, err error) {
	if !isArrayOrSlice(actual) {
		return false, fmt.Errorf("CyclicEqual matcher expects an array/slice.  Got:\n%s", format.Object(actual, 1))
	}
	expecteds := ce.Elements
	actuals := valuesOf(actual)

	if len(expecteds) == 0 {
		if len(actuals) == 0 {
			return true, nil
		}
		ce.lengthMismatch = true
		return false, nil
	}

	// if the cycle array has the same element at beginning and end, cut one off
	if len(actuals) > 1 {
		first := actuals[0]
		last := actuals[len(actuals)-1]
		succ, err := equals(first, last)
		if err != nil {
			return false, err
		}
		if succ {
			// first and last element are equal, cut last element off
			actuals = actuals[:len(actuals)-1]
		}
	}
	if len(expecteds) > 1 {
		first := expecteds[0]
		last := expecteds[len(expecteds)-1]
		succ, err := equals(first, last)
		if err != nil {
			return false, err
		}
		if succ {
			// first and last element are equal, cut last element off
			expecteds = expecteds[:len(expecteds)-1]
		}
	}

	if len(expecteds) != len(actuals) {
		ce.lengthMismatch = true
		return false, nil
	}

	// Idea: iterate over actual until the first element of expected is found.
	// Then iterate over both, comparing the elements. If the end of actual is reached,
	// continue from the beginning.
	// If any two elements don't match, the cycles are not identical.
	skew := 0
	for ; skew < len(actuals); skew++ {
		succ, err := equals(actuals[skew], expecteds[0])
		if err != nil {
			return false, err
		}
		if succ {
			break
		}
	}
	if skew == len(actuals) {
		ce.firstExpectedNotFound = true
		return false, nil
	}
	i := 1
	for ; i < len(expecteds); i++ {
		j := (skew + i) % len(actuals)
		succ, err := equals(actuals[j], expecteds[i])
		if err != nil {
			return false, err
		}
		if !succ {
			ce.first = expecteds[i-1]
			ce.actualSecond = actuals[j]
			ce.expectedSecond = expecteds[i]
			return false, nil
		}
	}

	return true, nil
}

func (ce *CyclicEqualMatcher) FailureMessage(actual interface{}) string {
	message := format.Message(actual, "to be cyclically equal to", presentable(ce.Elements))
	if ce.lengthMismatch {
		message = fmt.Sprintf("%s\nbut the amount of unique elements in both cycles is different", message)
	} else if ce.firstExpectedNotFound {
		message = fmt.Sprintf("%s\nbut the actual value is missing the first expected element\n%s", message, format.Object(ce.Elements[0], 1))
	} else {
		message = fmt.Sprintf("%s\nbut element\n%s\nis followed by element\n%s\nbut it was expected to be followed by element\n%s",
			message,
			format.Object(ce.first, 1),
			format.Object(ce.actualSecond, 1),
			format.Object(ce.expectedSecond, 1))
	}
	return message
}

func (ce *CyclicEqualMatcher) NegatedFailureMessage(actual interface{}) string {
	return format.Message(actual, "not to be cyclically equal to", presentable(ce.Elements))
}

// copied from gomega library
func isArrayOrSlice(a interface{}) bool {
	if a == nil {
		return false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

// copied from gomega library and modified
func valuesOf(actual interface{}) []interface{} {
	value := reflect.ValueOf(actual)
	values := []interface{}{}
	for i := 0; i < value.Len(); i++ {
		values = append(values, value.Index(i).Interface())
	}

	return values
}

// copied from gomega library
func presentable(elems []interface{}) interface{} {
	elems = flatten(elems)

	if len(elems) == 0 {
		return []interface{}{}
	}

	sv := reflect.ValueOf(elems)
	tt := sv.Index(0).Elem().Type()
	for i := 1; i < sv.Len(); i++ {
		if sv.Index(i).Elem().Type() != tt {
			return elems
		}
	}

	ss := reflect.MakeSlice(reflect.SliceOf(tt), sv.Len(), sv.Len())
	for i := 0; i < sv.Len(); i++ {
		ss.Index(i).Set(sv.Index(i).Elem())
	}

	return ss.Interface()
}

// copied from gomega library
func flatten(elems []interface{}) []interface{} {
	if len(elems) != 1 || !isArrayOrSlice(elems[0]) {
		return elems
	}

	value := reflect.ValueOf(elems[0])
	flattened := make([]interface{}, value.Len())
	for i := 0; i < value.Len(); i++ {
		flattened[i] = value.Index(i).Interface()
	}
	return flattened
}

func equals(actual, expected interface{}) (bool, error) {
	em := gomegamatchers.EqualMatcher{Expected: expected}
	succ, err := em.Match(actual)
	if err != nil {
		return succ, fmt.Errorf("error comparing elements %v and %v: %w", actual, expected, err)
	}
	return succ, nil
}
