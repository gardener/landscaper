// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/landscaper/pkg/utils/dependencies/queue"
)

type graph struct {
	edges map[string]sets.String

	// alreadyCheckedElements contains elements of which it is already known that they cannot reach a cycle.
	// The set is initially empty and grows during the algorithm.
	alreadyCheckedElements sets.String
}

// nodeWithPath stores a node of a graph as well as the path that led to this node.
// This is a helper struct for the breadth first search below.
type nodeWithPath struct {
	node string
	path []string
}

func (nwp nodeWithPath) hasVisitedBefore(elem string) bool {
	for _, p := range nwp.path {
		if p == elem {
			return true
		}
	}
	return false
}

func newGraph(edges map[string]sets.String) *graph {
	return &graph{
		edges:                  edges,
		alreadyCheckedElements: sets.NewString(),
	}
}

func (g *graph) hasCycle() (bool, []string) {
	graphNodes := queue.New[nodeWithPath]()
	for node := range g.edges {
		graphNodes.Append(nodeWithPath{
			node: node,
			path: []string{node},
		})
	}
	// start a breadth first search for cycles from all graph nodes simultaneously
	shortestCycle := g.breadthFirstSearchForCycles(graphNodes)
	return shortestCycle != nil, shortestCycle
}

// breadthFirstSearchForCycles searches the graph for cycles.
// It returns the first cycle found.
// If no cycle is found, nil is returned.
func (g *graph) breadthFirstSearchForCycles(todo queue.Queue[nodeWithPath]) []string {
	for !todo.IsEmpty() {
		cur, _ := todo.Pop()
		successors := g.edges[cur.node]
		for _, succ := range successors.List() {
			newPath := append(cur.path, succ)
			if cur.hasVisitedBefore(succ) {
				return newPath
			}
			todo.Append(nodeWithPath{
				node: succ,
				path: newPath,
			})
		}
	}
	return nil
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
