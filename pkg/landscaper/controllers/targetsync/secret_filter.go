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

type secretFilter struct {
	secretNameExpression string
	compiledExpression   *regexp.Regexp
}

var _ predicate.Predicate = &secretFilter{}

func newSecretFilter(secretNameExpression string) (*secretFilter, error) {
	compiledExpression, err := regexp.Compile(secretNameExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression to filter secrets: %s", secretNameExpression)
	}

	return &secretFilter{
		secretNameExpression: secretNameExpression,
		compiledExpression:   compiledExpression,
	}, nil
}

func (p *secretFilter) shouldBeProcessed(obj client.Object) bool {
	return p.compiledExpression.MatchString(obj.GetName())
}

// Create returns true if the Create event should be processed
func (p *secretFilter) Create(event event.CreateEvent) bool {
	return p.shouldBeProcessed(event.Object)
}

// Delete returns true if the Delete event should be processed
func (p *secretFilter) Delete(event event.DeleteEvent) bool {
	return p.shouldBeProcessed(event.Object)
}

// Update returns true if the Update event should be processed
func (p *secretFilter) Update(event event.UpdateEvent) bool {
	return p.shouldBeProcessed(event.ObjectNew)
}

// Generic returns true if the Generic event should be processed
func (p *secretFilter) Generic(event event.GenericEvent) bool {
	return p.shouldBeProcessed(event.Object)
}
