// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import "fmt"

// Requirements is a helper struct for object methods which require other methods to have been called before.
type Requirements map[string]*Requirement

// NewRequirements retuns a new Requirements object.
func NewRequirements() Requirements {
	return map[string]*Requirement{}
}

// Requirement represents a requirement, consisting of a 'Satisfy' method and a boolean to check whether it has been called.
type Requirement struct {
	Satisfy   func() error
	satisfied bool
}

// IsSatisfied returns whether the requirement is satisfied (= its 'Satisfy' method has been called before)
func (r *Requirement) IsSatisfied() bool {
	return r.satisfied
}

// Register registers a new requirement. It takes an identifer and a function which, when called, will satisfy the requirement.
func (rs Requirements) Register(id string, satisfy func() error) {
	rs[id] = &Requirement{Satisfy: satisfy}
}

// Require checks if all referenced requirements have been satisfied.
// It returns an error if any of the given IDs doesn't match a requirement which has been registered before.
// For all requirements, it is checked whether the requirement is already satisfied.
//   If not, its 'Satisfy' method is called.
//   If 'Satisfy' returns without errors, the requirement is marked as satisfied,
//   otherwise the error is returned without checking further requirements.
func (rs Requirements) Require(ids ...string) error {
	for _, id := range ids {
		req, ok := rs[id]
		if !ok {
			return NewRequirementError(id, fmt.Errorf("unknown requirement %q", id))
		}
		if !req.IsSatisfied() {
			err := req.Satisfy()
			if err != nil {
				return NewRequirementError(id, err)
			}
			req.satisfied = true
		}
	}
	return nil
}

// HasRequirement returns whether a requirement with the given ID has been registered.
func (rs Requirements) HasRequirement(id string) bool {
	_, ok := rs[id]
	return ok
}

// IsSatisfied returns true if a requirement with the given ID has been registered and satisfied.
func (rs Requirements) IsSatisfied(id string) bool {
	req, ok := rs[id]
	return ok && req.IsSatisfied()
}

// SetSatisfied allows to manually satisfy (or 'unsatisfy') a requirement.
func (rs Requirements) SetSatisfied(id string, satisfied bool) error {
	req, ok := rs[id]
	if !ok {
		return NewRequirementError(id, fmt.Errorf("unknown requirement %q", id))
	}
	req.satisfied = satisfied
	return nil
}

type RequirementError struct {
	// Requirement is the name of the requirement which caused the error
	Requirement string
	// Error is the error
	Err error
}

func NewRequirementError(req string, err error) RequirementError {
	return RequirementError{
		Requirement: req,
		Err:         err,
	}
}

func IsRequirementError(err error) bool {
	_, ok := err.(RequirementError)
	return ok
}

func (re RequirementError) Error() string {
	return re.Err.Error()
}
