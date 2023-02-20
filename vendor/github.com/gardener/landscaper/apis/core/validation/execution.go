// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/apis/core"
)

// ValidateExecution validates an Execution
func ValidateExecution(execution *core.Execution) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateExecutionSpec(field.NewPath("spec"), execution.Spec)...)
	return allErrs
}

// ValidateExecutionSpec validtes the spec of a execution object
func ValidateExecutionSpec(fldpath *field.Path, spec core.ExecutionSpec) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateDeployItemTemplateList(fldpath.Child("deployItems"), spec.DeployItems)...)
	return allErrs
}

// ValidateDeployItemTemplateList validates a list of deploy item templates.
func ValidateDeployItemTemplateList(fldPath *field.Path, list core.DeployItemTemplateList) field.ErrorList {
	allErrs := field.ErrorList{}
	names := sets.NewString()
	hasDuplicates := false
	for i, tmpl := range list {
		tmplPath := fldPath.Index(i)
		if len(tmpl.Name) != 0 {
			if names.Has(tmpl.Name) {
				allErrs = append(allErrs, field.Duplicate(tmplPath, tmpl.Name))
				hasDuplicates = true
			}
			names.Insert(tmpl.Name)
			tmplPath = tmplPath.Key(tmpl.Name)
		}
		allErrs = append(allErrs, ValidateDeployItemTemplate(tmplPath, tmpl)...)
	}

	if !hasDuplicates { // cycle check identifies items by name and the behaviour is undefined if duplicate items are present
		done := sets.NewString()
		cycles := []Cycle{}
		undefined := []UndefinedDeployItemReference{}
		for i := range list {
			c, u, d := getCyclesAndUndefinedDependencies(list, i, []string{}, done)
			cycles = append(cycles, c...)
			undefined = append(undefined, u...)
			done.Insert(d.UnsortedList()...)
		}
		for _, c := range cycles {
			allErrs = append(allErrs, field.Invalid(fldPath, c, "cycle found in dependencies"))
		}
		for _, u := range undefined {
			idx, _ := getDeployItemTemplateByName(list, u.Source)
			allErrs = append(allErrs, field.Invalid(fldPath.Index(idx).Key(u.Source), u.Target, "depends on undefined deploy item"))
		}
	}

	return allErrs
}

// takes arguments:
// 1. a list of deploy item templates
// 2. the index of the deploy item template that should be validated
// 3. a list of already visited deploy item templates (referenced by name), shows the current state of the depth-first search
// 4. a set of deploy item templates (referenced by name) that have already been checked and can be ignored
// returns:
// 1. a list of Cycle objects, representing found cyclic dependencies
// 2. a list of UndefinedDeployItemReference objects, representing found dependencies to undefined deploy items
// 3. a set of all deploy item templates (referenced by name) that have already been checked (necessary to avoid finding the same cycle multiple times)
func getCyclesAndUndefinedDependencies(list core.DeployItemTemplateList, index int, visited visitedList, done sets.String) ([]Cycle, []UndefinedDeployItemReference, sets.String) { //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
	current := list[index]
	cycles := []Cycle{}
	undefined := []UndefinedDeployItemReference{}
	if done.Has(current.Name) { // current deploy item has already been handled
		return cycles, undefined, done
	}
	done.Insert(current.Name)
	if len(current.DependsOn) == 0 { // current deploy item doesn't depend on anything
		done.Insert(current.Name)
		return cycles, undefined, done
	}
	visitedNew := append(visited, current.Name)
	for _, do := range current.DependsOn {
		cycle := visitedNew.getCycle(do)
		if cycle != nil {
			cycles = append(cycles, cycle)
			continue
		}
		idx, _ := getDeployItemTemplateByName(list, do)
		if idx < 0 {
			// current deploy item depends on an undefined deploy item
			undefined = append(undefined, UndefinedDeployItemReference{
				Source: current.Name,
				Target: do,
			})
			continue
		}
		resAddC, resAddU, newDone := getCyclesAndUndefinedDependencies(list, idx, visitedNew, done)
		cycles = append(cycles, resAddC...)
		undefined = append(undefined, resAddU...)
		done.Insert(newDone.UnsortedList()...)
	}
	return cycles, undefined, done
}

// Cycle represents a cycle of dependencies
type Cycle []string

// UndefinedDeployItemReference represents a dependency to an undefined deploy item
type UndefinedDeployItemReference struct {
	Source string
	Target string
}

// visitedList represents a list of already visited elements during a depth-first search in a graph
type visitedList []string

// getCycle returns a Cycle object containg the elements of the cycle, if a cycle is found, and nil otherwise
// basically checks whether the given argument is already present in the list and returns all elements from there up to the end of the list if so
func (visited visitedList) getCycle(new string) Cycle {
	for i, e := range visited { // check if current dependency has already been visited
		if new == e {
			// cycle found, identify elements of cycle
			cycle := []string{}
			for j := i; j < len(visited); j++ {
				cycle = append(cycle, visited[j])
			}
			return cycle
		}
	}
	return nil
}

// Given a DeployItemTemplateList and a name, this function returns the index and the DeployItemTemplate with that name.
// If no element with that name exists, the returned index is -1 and the returned DeployItemTemplate is undefined.
func getDeployItemTemplateByName(list core.DeployItemTemplateList, name string) (int, core.DeployItemTemplate) {
	resIndex := -1
	var resDit core.DeployItemTemplate = core.DeployItemTemplate{}
	for idx, elem := range list {
		if elem.Name == name {
			resIndex = idx
			resDit = elem
			break
		}
	}
	return resIndex, resDit
}

// ValidateDeployItemTemplate validates a deploy item template.
func ValidateDeployItemTemplate(fldPath *field.Path, tmpl core.DeployItemTemplate) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(tmpl.Name) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name must not be empty"))
	}

	if len(tmpl.Type) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "type must not be empty"))
	}

	if tmpl.Target != nil {
		allErrs = append(allErrs, ValidateObjectReference(*tmpl.Target, fldPath.Child("target"))...)
	}

	if len(tmpl.Labels) != 0 {
		allErrs = append(allErrs, metav1validation.ValidateLabels(tmpl.Labels, fldPath.Child("labels"))...)
	}

	return allErrs
}
