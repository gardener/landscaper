// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"fmt"
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type nameFilter struct {
	nameExpression     string
	compiledExpression *regexp.Regexp
}

var _ predicate.Predicate = &nameFilter{}

func newNameFilter(nameExpression string) (*nameFilter, error) {
	if nameExpression == "*" {
		nameExpression = ".*"
	}

	compiledExpression, err := regexp.Compile(nameExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression to filter names: %s", nameExpression)
	}

	return &nameFilter{
		nameExpression:     nameExpression,
		compiledExpression: compiledExpression,
	}, nil
}

func (p *nameFilter) shouldBeProcessed(obj client.Object) bool {
	return p.compiledExpression.MatchString(obj.GetName())
}

// Create returns true if the Create event should be processed
func (p *nameFilter) Create(event event.CreateEvent) bool {
	return p.shouldBeProcessed(event.Object)
}

// Delete returns true if the Delete event should be processed
func (p *nameFilter) Delete(event event.DeleteEvent) bool {
	return p.shouldBeProcessed(event.Object)
}

// Update returns true if the Update event should be processed
func (p *nameFilter) Update(event event.UpdateEvent) bool {
	return p.shouldBeProcessed(event.ObjectNew)
}

// Generic returns true if the Generic event should be processed
func (p *nameFilter) Generic(event event.GenericEvent) bool {
	return p.shouldBeProcessed(event.Object)
}
