package dependencies

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
)

type graph struct {
	edges map[string]sets.String

	// alreadyCheckedElements contains elements of which it is already known that they cannot reach a cycle.
	// The set is initially empty and grows during the algorithm.
	alreadyCheckedElements sets.String
}

func newGraph(edges map[string]sets.String) *graph {
	return &graph{
		edges:                  edges,
		alreadyCheckedElements: sets.NewString(),
	}
}

func (g *graph) hasCycle() (bool, []string) {
	for node := range g.edges {
		visited := make(map[string]bool, 0)
		hasCycle, cycle := g.canReachCycle(node, visited)
		if hasCycle {
			return true, append(cycle, node)
		}
	}

	return false, nil
}

// canReachCycle returns true if starting from the given node one can reach an element of a cycle.
// Assumptions: 1. from each visited element there exists a path to the given node, and
// 2. the node itself is not a visited element.
func (g *graph) canReachCycle(node string, visited map[string]bool) (bool, []string) {
	if g.alreadyCheckedElements.Has(node) {
		return false, nil
	}

	visited[node] = true

	successors := g.edges[node]
	for succ := range successors {
		if visited[succ] {
			return true, []string{succ}
		} else {
			hasCycle, cycle := g.canReachCycle(succ, visited)
			if hasCycle {
				return true, append(cycle, succ)
			}
		}
	}

	visited[node] = false

	g.alreadyCheckedElements.Insert(node)

	return false, nil
}

func (g *graph) getReverseOrder() ([]string, error) {
	hasCycle, cycle := g.hasCycle()

	if hasCycle {
		return nil, fmt.Errorf("graph has cycle: %s", cycle)
	}

	result := []string{}
	addedNodes := make(map[string]bool, 0)
	notAddedNodes := []string{}

	for k := range g.edges {
		notAddedNodes = append(notAddedNodes, k)
	}

	for len(notAddedNodes) > 0 {
		newNotAddedNodes := []string{}

		for _, nextNode := range notAddedNodes {
			if g.allSuccessorAlreadyAdded(nextNode, addedNodes) {
				addedNodes[nextNode] = true
				result = append(result, nextNode)
			} else {
				newNotAddedNodes = append(newNotAddedNodes, nextNode)
			}
		}

		notAddedNodes = newNotAddedNodes
	}

	return result, nil
}

func (g *graph) allSuccessorAlreadyAdded(node string, alreadyAddedNodes map[string]bool) bool {
	succs := g.edges[node]
	for _, nextSucc := range succs.List() {
		if _, ok := alreadyAddedNodes[nextSucc]; !ok {
			return false
		}
	}

	return true
}
