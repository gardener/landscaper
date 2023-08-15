// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"strings"
)

// ErrorList is a helper struct for situations in which multiple errors should be returned as a single one.
type ErrorList struct {
	Errs []error
}

// NewErrorList creates a new ErrorList containing the provided errors.
func NewErrorList(errs ...error) *ErrorList {
	res := &ErrorList{
		Errs: []error{},
	}
	return res.Append(errs...)
}

// Aggregate aggregates all errors in the ErrorList into a single error.
// Returns nil if the ErrorList is either nil or empty.
// If the list contains a single error, that error is returned.
// Otherwise, a new error is constructed by appending all contained errors' messages.
func (el *ErrorList) Aggregate() error {
	if el == nil || len(el.Errs) == 0 {
		return nil
	} else if len(el.Errs) == 1 {
		return el.Errs[0]
	}
	sb := strings.Builder{}
	sb.WriteString("multiple errors occurred:")
	for _, e := range el.Errs {
		sb.WriteString("\n")
		sb.WriteString(e.Error())
	}
	return errors.New(sb.String())
}

// Append appends all given errors to the ErrorList.
// This modifies the receiver object.
// nil pointers in the arguments are ignored.
// Returns the receiver for chaining.
func (el *ErrorList) Append(errs ...error) *ErrorList {
	for _, e := range errs {
		if e != nil {
			el.Errs = append(el.Errs, e)
		}
	}
	return el
}
