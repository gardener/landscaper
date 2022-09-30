package dependencies

import (
	"fmt"

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

	predecessors, _ := instNode.fetchPredecessors(otherNodes, false)
	return predecessors
}

func CheckForCyclesAndDuplicateExports(instTemplates []*lsv1alpha1.InstallationTemplate, computeOrder bool) ([]*lsv1alpha1.InstallationTemplate, error) {
	instNodes := []*installationNode{}

	for _, next := range instTemplates {
		instNodes = append(instNodes, newInstallationNodeFromInstallationTemplate(next))
	}

	edges := map[string]sets.String{}
	for _, next := range instNodes {
		predecessors, hasDublicateExports := next.fetchPredecessors(instNodes, true)
		if hasDublicateExports {
			return nil, fmt.Errorf("the installation %s gets imports with same name from different predecessors", next.name)
		}
		edges[next.name] = predecessors
	}

	g := newGraph(edges)

	hasCycle, cycle := g.hasCycle()

	if hasCycle {
		return nil, fmt.Errorf("the subinstallations have a cycle: %s", cycle)
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

func (r *installationNode) fetchPredecessors(otherNodes []*installationNode, checkDuplicateExports bool) (sets.String, bool) {
	dataExports, targetExports, hasDuplicateExports := r.getExportMaps(otherNodes, checkDuplicateExports)

	if hasDuplicateExports {
		return nil, hasDuplicateExports
	}

	predecessors := sets.NewString()
	for _, imp := range r.imports.Data {
		if len(imp.DataRef) == 0 {
			// only dataRef imports can refer to sibling exports
			continue
		}
		source, ok := dataExports[imp.DataRef]
		if ok {
			// no sibling exports this import, it has to come from the parent
			// this is already checked by validation, no need to verify it here
			predecessors.Insert(source)
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
			source, ok := targetExports[target]
			if !ok {
				// no sibling exports this import, it has to come from the parent
				// this is already checked by validation, no need to verify it here
				continue
			}
			predecessors.Insert(source)
		}
	}

	return predecessors, hasDuplicateExports
}

func (r *installationNode) getExportMaps(otherNodes []*installationNode, checkDuplicateExports bool) (map[string]string, map[string]string, bool) {
	dataExports := map[string]string{}
	targetExports := map[string]string{}
	hasDuplicateExports := false

	for _, sibling := range otherNodes {
		// if called with otherNodes including the current one (this happens sometimes)
		if sibling.name == r.name {
			continue
		}
		for _, exp := range sibling.exports.Data {
			_, ok := dataExports[exp.DataRef]
			if checkDuplicateExports && ok {
				return nil, nil, true
			}
			dataExports[exp.DataRef] = sibling.name
		}
		for _, exp := range sibling.exports.Targets {
			_, ok := targetExports[exp.Target]
			if checkDuplicateExports && ok {
				return nil, nil, true
			}
			targetExports[exp.Target] = sibling.name
		}
	}

	return dataExports, targetExports, hasDuplicateExports
}
