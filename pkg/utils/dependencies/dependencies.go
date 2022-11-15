// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dependencies

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func FetchPredecessorsFromInstallation(installation *lsv1alpha1.Installation,
	otherInstallations []*lsv1alpha1.Installation) sets.String {
	instNode := newInstallationNodeFromInstallation(installation)

	otherNodes := []*installationNode{}
	for _, next := range otherInstallations {
		otherNodes = append(otherNodes, newInstallationNodeFromInstallation(next))
	}

	predecessors, _ := instNode.fetchPredecessors(otherNodes)
	return predecessors
}

func CheckForCyclesAndDuplicateExports(instTemplates []*lsv1alpha1.InstallationTemplate, computeOrder bool) ([]*lsv1alpha1.InstallationTemplate, error) {
	instNodes := []*installationNode{}

	for _, next := range instTemplates {
		instNodes = append(instNodes, newInstallationNodeFromInstallationTemplate(next))
	}

	edges := map[string]sets.String{}
	for _, next := range instNodes {
		predecessors, err := next.fetchPredecessors(instNodes)
		if err != nil {
			return nil, err
		}
		edges[next.name] = predecessors
	}

	g := newGraph(edges)

	hasCycle, cycle := g.hasCycle()

	if hasCycle {
		return nil, fmt.Errorf("the subinstallations have a cycle: %s", strings.Join(cycle, " -{depends_on}-> "))
	}

	if computeOrder {
		orderedNodes, _ := g.getReverseOrder()

		orderedTemplates := []*lsv1alpha1.InstallationTemplate{}

		templatesMap := map[string]*lsv1alpha1.InstallationTemplate{}
		for _, next := range instTemplates {
			templatesMap[next.Name] = next
		}

		for _, next := range orderedNodes {
			orderedTemplates = append(orderedTemplates, templatesMap[next])
		}

		return orderedTemplates, nil

	} else {
		return nil, nil
	}
}

type installationNode struct {
	name    string
	exports lsv1alpha1.InstallationExports
	imports lsv1alpha1.InstallationImports
}

func newInstallationNodeFromInstallation(installation *lsv1alpha1.Installation) *installationNode {
	return &installationNode{
		name:    installation.Name,
		exports: installation.Spec.Exports,
		imports: installation.Spec.Imports,
	}
}

func newInstallationNodeFromInstallationTemplate(installation *lsv1alpha1.InstallationTemplate) *installationNode {
	return &installationNode{
		name:    installation.Name,
		exports: installation.Exports,
		imports: installation.Imports,
	}
}

func (r *installationNode) fetchPredecessors(otherNodes []*installationNode) (sets.String, error) {
	dataExports, targetExports, hasDuplicateExports := r.getExportMaps(otherNodes)

	if hasDuplicateExports {
		msg := strings.Builder{}
		msg.WriteString("the following exports are exported by multiple nested installations:")
		dupExpFound := false
		for exp, sources := range dataExports {
			if sources.Len() > 1 {
				if !dupExpFound {
					dupExpFound = true
					msg.WriteString("\n  data exports:")
				}
				msg.WriteString(fmt.Sprintf("\n    '%s' is exported by [%s]", exp, strings.Join(sources.List(), ", ")))
			}
		}
		dupExpFound = false
		for exp, sources := range targetExports {
			if sources.Len() > 1 {
				if !dupExpFound {
					dupExpFound = true
					msg.WriteString("\n  target exports:")
				}
				msg.WriteString(fmt.Sprintf("\n    '%s' is exported by [%s]", exp, strings.Join(sources.List(), ", ")))
			}
		}
		return nil, errors.New(msg.String())
	}

	predecessors := sets.NewString()
	for _, imp := range r.imports.Data {
		if len(imp.DataRef) == 0 {
			// only dataRef imports can refer to sibling exports
			continue
		}
		sources, ok := dataExports[imp.DataRef]
		if ok {
			// no sibling exports this import, it has to come from the parent
			// this is already checked by validation, no need to verify it here
			predecessors.Insert(sources.UnsortedList()...)
		}
	}

	for _, imp := range r.imports.Targets {
		targets := []string{}

		if len(imp.Target) != 0 {
			targets = append(targets, imp.Target)
		} else if len(imp.Targets) != 0 {
			targets = imp.Targets
		} else {
			// targetListReferences can only refer to parent imports, not to sibling exports
			continue
		}

		for _, target := range targets {
			sources, ok := targetExports[target]
			if !ok {
				// no sibling exports this import, it has to come from the parent
				// this is already checked by validation, no need to verify it here
				continue
			}
			predecessors.Insert(sources.UnsortedList()...)
		}
	}

	return predecessors, nil
}

// getExportMaps returns a mapping from sibling export names to the exporting siblings' names.
// If for any given key the length of its value (a set) is greater than 1, this means that two or more siblings define the same export.
// The third returned parameter indicates whether this has happened or not (true in case of duplicate exports).
func (r *installationNode) getExportMaps(otherNodes []*installationNode) (map[string]sets.String, map[string]sets.String, bool) {
	dataExports := map[string]sets.String{}
	targetExports := map[string]sets.String{}
	hasDuplicateExports := false

	for _, sibling := range otherNodes {
		// if called with otherNodes including the current one (this happens sometimes)
		if sibling.name == r.name {
			continue
		}
		for _, exp := range sibling.exports.Data {
			var de sets.String
			var ok bool
			de, ok = dataExports[exp.DataRef]
			if !ok {
				de = sets.NewString()
			}
			de.Insert(sibling.name)
			if !hasDuplicateExports && de.Len() > 1 {
				hasDuplicateExports = true
			}
			dataExports[exp.DataRef] = de
		}
		for _, exp := range sibling.exports.Targets {
			var te sets.String
			var ok bool
			te, ok = targetExports[exp.Target]
			if !ok {
				te = sets.NewString()
			}
			te.Insert(sibling.name)
			if !hasDuplicateExports && te.Len() > 1 {
				hasDuplicateExports = true
			}
			targetExports[exp.Target] = te
		}
	}

	return dataExports, targetExports, hasDuplicateExports
}
