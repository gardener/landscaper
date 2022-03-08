// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

type DependencyCycle struct {
	origin *dependencyCycleElement
	last   *dependencyCycleElement
	size   int
}

type dependencyCycleElement struct {
	name        string                  // name of the importing entity
	importNames sets.String             // names of the imports which cause the dependency
	dependsOn   *dependencyCycleElement // where the import comes from
}

// copy copies the given element. The dependsOn pointer is set to nil and not copied.
func (dce *dependencyCycleElement) copy() *dependencyCycleElement {
	res := &dependencyCycleElement{
		name: dce.name,
	}
	res.importNames = copyStringSet(dce.importNames)
	return res
}

// NewDependencyCycle starts a new dependency cycle. It must be initialized with one element.
// Internally, it works like a linked list, except that it is expected to form a cycle at some point.
func NewDependencyCycle(templateName string) *DependencyCycle {
	dc := &DependencyCycle{
		origin: &dependencyCycleElement{
			name: templateName,
		},
	}
	dc.last = dc.origin
	return dc
}

// Add adds a new dependency to the cycle. It returns a bool indicating whether the cycle is closed:
// it is true if the given name matches the name of the origin element of the cycle.
// No-op if the cycle is already closed.
func (dc *DependencyCycle) Add(templateName string, importNames sets.String) bool {
	if dc.IsClosed() {
		// cycle is already closed, can't add to it
		return true
	}

	dc.last.importNames = importNames
	if templateName == dc.origin.name {
		dc.last.dependsOn = dc.origin
		return true
	}
	dc.last.dependsOn = &dependencyCycleElement{
		name: templateName,
	}
	dc.last = dc.last.dependsOn
	dc.size++
	return false
}

// Copy copies the dependency cycle.
func (dc *DependencyCycle) Copy() *DependencyCycle {
	res := &DependencyCycle{}
	res.origin = dc.origin.copy()
	res.size = dc.size
	originCurrent := dc.origin.dependsOn
	newLast := res.origin
	for originCurrent != nil {
		newLast.dependsOn = originCurrent.copy()
		originCurrent = originCurrent.dependsOn
		newLast = newLast.dependsOn
		if originCurrent == dc.origin {
			// end of closed cycle reached
			break
		}
	}
	if originCurrent == nil {
		// only set last if the cycle is not closed
		res.last = newLast
	}
	return res
}

// IsClosed returns whether the cycle is closed.
func (dc *DependencyCycle) IsClosed() bool {
	return dc.last == nil
}

// Size returns the number of elements in this cycle.
// Note that closing the cycle - by adding its origin element to it - does not increase the size.
func (dc *DependencyCycle) Size() int {
	return dc.size
}

// List returns the names of all templates which are part of this cycle.
func (dc *DependencyCycle) List() []string {
	current := dc.origin
	res := make([]string, 0, dc.size)
	for current != nil {
		res = append(res, current.name)
		current = current.dependsOn
		if current == dc.origin {
			break
		}
	}
	return res
}

// Has returns true if the given name is part of the cycle.
func (dc *DependencyCycle) Has(name string) bool {
	current := dc.origin
	for current != nil {
		if current.name == name {
			return true
		}
		current = current.dependsOn
		if current == dc.origin {
			break
		}
	}
	return false
}

// Last returns the name of the last element of the cycle.
// If the cycle is closed, it returns an empty string.
func (dc *DependencyCycle) Last() string {
	if dc.last != nil {
		return dc.last.name
	}
	return ""
}

// Origin returns the name of the origin element of the cycle.
func (dc *DependencyCycle) Origin() string {
	return dc.origin.name
}

// Close adds an element which has the same name as the origin, closing the cycle.
// Returns the cycle for chaining
func (dc *DependencyCycle) Close(importNames sets.String) *DependencyCycle {
	if !dc.IsClosed() {
		dc.Add(dc.origin.name, importNames)
	}
	return dc
}

type RelationshipTuple struct {
	Exporting string
	Importing string
}

// ImportRelationships maps tuples of entities to the set of im-/exports which connect them
type ImportRelationships map[RelationshipTuple]sets.String

func (ir ImportRelationships) Get(exporting, importing string) (sets.String, bool) {
	tmp := RelationshipTuple{
		Exporting: exporting,
		Importing: importing,
	}
	imps, ok := ir[tmp]
	return imps, ok
}

func (ir ImportRelationships) Add(exporting, importing, imp string) ImportRelationships {
	tmp := RelationshipTuple{
		Exporting: exporting,
		Importing: importing,
	}
	if _, ok := ir[tmp]; !ok {
		ir[tmp] = sets.NewString()
	}
	ir[tmp].Insert(imp)
	return ir
}

// Equal returns true if both objects describe the same cycle
// meaning the same template names in the same order.
// Cycles which are not closed are only equal if they have the same origin,
// for closed cycles, this doesn't matter.
// If only one of the cycles is closed, they are considered to be not equal.
// The actual cycle must not be nil, if the given cycle is nil, they are considered to be not equal.
func (dc *DependencyCycle) Equal(other *DependencyCycle) bool {
	if other == nil {
		// we assume that dc is not nil
		return false
	}
	otherOrigin := other.origin
	if dc.IsClosed() {
		if !other.IsClosed() {
			return false
		}
		if dc.origin.name != otherOrigin.name {
			// find matching element, if any, in case of different origins
			for tmp := otherOrigin.dependsOn; tmp != otherOrigin; tmp = tmp.dependsOn {
				if tmp.name == dc.origin.name {
					// found matching element
					otherOrigin = tmp
					break
				}
			}
		}
	} else {
		if other.IsClosed() {
			return false
		}
	}
	if dc.origin.name != otherOrigin.name {
		// no matching origin found, cycles cannot be equal
		return false
	}
	for thisCurrent, otherCurrent := dc.origin.dependsOn, otherOrigin.dependsOn; thisCurrent != dc.origin && otherCurrent != otherOrigin; thisCurrent, otherCurrent = thisCurrent.dependsOn, otherCurrent.dependsOn {
		if thisCurrent.name != otherCurrent.name {
			return false
		}
		if thisCurrent.importNames != nil {
			if !thisCurrent.importNames.Equal(otherCurrent.importNames) {
				return false
			}
		} else {
			if otherCurrent.importNames != nil {
				return false
			}
		}
	}
	return true
}

func (dc *DependencyCycle) String() string {
	return dc.StringWithSeparator(" -> ", " -", "-> ", "[", ", ", "]")
}

func (dc *DependencyCycle) StringWithSeparator(sep, sepBeforeImports, sepAfterImports, importStart, importSep, importEnd string) string {
	var sb strings.Builder
	current := dc.origin
	for current != nil {
		sb.WriteString(current.name)
		if current.dependsOn != nil {
			if current.importNames != nil {
				sb.WriteString(sepBeforeImports)
				sb.WriteString(importStart)
				sb.WriteString(strings.Join(current.importNames.List(), importSep))
				sb.WriteString(importEnd)
				sb.WriteString(sepAfterImports)
			} else {
				sb.WriteString(sep)
			}
		}
		current = current.dependsOn
		if current == dc.origin {
			sb.WriteString(current.name)
			break
		}
	}
	return sb.String()
}

// DetermineCyclicDependencyDetails takes a list of installation templates which all depend on each other in a cyclic manner
// and a mapping from template names to the names of the templates they depend on.
// Optionally, the import relationships (which two elements are connected by which imports) can be given.
// It returns a list of found cycles.
func DetermineCyclicDependencyDetails(elements sets.String, dependencies map[string]sets.String, impRel ImportRelationships) []*DependencyCycle {
	res := []*DependencyCycle{}
	elemList := elements.List()

	// All cycles which contain elem will be found with a call to determineCycles(nil, elem, ...)
	// To also find all cycles which don't contain elem, we remove elem from the set of elements and then repeat the process
	for i := 0; i < len(elemList); i++ {
		res = append(res, determineCycles(nil, elemList[i], sets.NewString().Insert(elemList[i+1:]...), dependencies, impRel)...)
	}

	return res
}

// determineCycles tries to add another element to the given cycle.
// The arguments are:
// - cycle: the current cycle. May be nil, then a new one will be created.
// - current: the current element.
// - elements: the set of potential elements. It is expected to contain only elements which are not already part of the cycle.
// - dependencies: a mapping from elements to all elements they depend on
// - impRel: optional mapping from im-/export relationships to the im-/exports causing the relationship
func determineCycles(cycle *DependencyCycle, current string, elements sets.String, dependencies map[string]sets.String, impRel ImportRelationships) []*DependencyCycle {
	res := []*DependencyCycle{}
	if len(current) == 0 {
		return res
	}
	if cycle == nil {
		cycle = NewDependencyCycle(current)
	} else {
		// fetch connecting imports, if given
		var imps sets.String
		if impRel != nil {
			imps, _ = impRel.Get(current, cycle.Last())
		}

		// add current element to cycle
		if cycle.Add(current, imps) {
			// cycle is closed
			// this case should not occur, since it would be catched before by the check below
			res = append(res, cycle)
			return res
		}
	}

	// check if we can close the cycle
	if dependencies[current].Has(cycle.Origin()) {
		// the current element, which was just added to the cycle, depends on the origin of the cycle
		// => add the origin again to close the cycle and add it to the result
		var imps sets.String
		if impRel != nil {
			imps, _ = impRel.Get(cycle.Origin(), current)
		}
		res = append(res, cycle.Copy().Close(imps))
	}

	for elem := range elements {
		if dependencies[current].Has(elem) {
			// we want to ignore all elements which are already part of the cycle
			// so let's build a new element list and remove the element we just added to the cycle
			newElements := copyStringSetWithoutElement(elements, current)
			// recursion
			// copy cycle to not alter the original
			res = append(res, determineCycles(cycle.Copy(), elem, newElements, dependencies, impRel)...)
		}
	}
	return res
}

func copyStringSet(set sets.String) sets.String {
	if set == nil {
		return nil
	}
	res := sets.NewString()
	for elem := range set {
		res.Insert(elem)
	}
	return res
}
func copyStringSetWithoutElement(set sets.String, element string) sets.String {
	if set == nil {
		return nil
	}
	res := sets.NewString()
	for elem := range set {
		if elem == element {
			continue
		}
		res.Insert(elem)
	}
	return res
}
