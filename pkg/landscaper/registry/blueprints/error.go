// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry

import (
	"fmt"
)

// ErrorReason represents a ociRegistry specific issue
type ErrorReason string

const (
	// WrongType is an error that is thrown when the component request is of the wrong format
	WrongType ErrorReason = "WrongType"

	// ComponentNotFound is an error that is thrown when the requested component cannot be found
	ComponentNotFound ErrorReason = "ComponentNotFound"

	// VersionNotFound is an error that is thrown when the requested component version cannot be found
	VersionNotFound ErrorReason = "VersionNotFound"

	// VersionParseError is an error that is thrown when a component's version cannot be parsed
	VersionParseError ErrorReason = "VersionParseError"

	// NotFound is an generic error that is thrown when the requested resource cannot be found
	NotFound ErrorReason = "NotFound"
)

type registryError struct {
	Reason  ErrorReason
	Message string
	Err     error
}

func (e *registryError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap implements the golang Unwrap function
func (e *registryError) Unwrap() error { return e.Err }

// NewComponentNotFoundError creates a new ComponentNotFoundError
func NewComponentNotFoundError(name string, err error) error {
	return &registryError{
		Reason:  ComponentNotFound,
		Message: fmt.Sprintf("The requested component %s cannot be found", name),
		Err:     err,
	}
}

// IsWrongTypeError checks if the error is a WrongType error
func IsWrongTypeError(err error) bool {
	return IsErrorForReason(err, ComponentNotFound)
}

// NewWrongTypeError creates a new WrongType error
func NewWrongTypeError(ttype, name, version string, err error) error {
	return &registryError{
		Reason:  WrongType,
		Message: fmt.Sprintf("The requested resource with name %s and version %s was of a wrong type %s", version, name, ttype),
		Err:     err,
	}
}

// IsComponentNotFoundError checks if the error is a ComponentNotFoundError
func IsComponentNotFoundError(err error) bool {
	return IsErrorForReason(err, ComponentNotFound)
}

// NewVersionNotFoundError creates a new ComponentNotFoundError
func NewVersionNotFoundError(name, version string, err error) error {
	return &registryError{
		Reason:  VersionNotFound,
		Message: fmt.Sprintf("The requested version %s for component %s cannot be found", version, name),
		Err:     err,
	}
}

// IsVersionNotFoundError checks if the error is a VersionNotFound
func IsVersionNotFoundError(err error) bool {
	return IsErrorForReason(err, VersionNotFound)
}

// NewVersionParseError creates a new VersionParseError
func NewVersionParseError(version string, err error) error {
	return &registryError{
		Reason:  VersionParseError,
		Message: fmt.Sprintf("The requested version %s cannot be parsed", version),
		Err:     err,
	}
}

// IsVersionParseError checks if the error is a VersionParseError
func IsVersionParseError(err error) bool {
	return IsErrorForReason(err, VersionParseError)
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(name string, err error) error {
	return &registryError{
		Reason:  NotFound,
		Message: fmt.Sprintf("The requested component %s cannot be found", name),
		Err:     err,
	}
}

// IsNotFoundError checks if the error is either a ComponentNotFoundError or a VersionNotFoundError or a generic not found error
func IsNotFoundError(err error) bool {
	return IsComponentNotFoundError(err) || IsVersionNotFoundError(err) || IsErrorForReason(err, NotFound)
}

// IsErrorForReason checks if the error is a ociRegistry error and of the givne reason.
func IsErrorForReason(err error, reason ErrorReason) bool {
	if err == nil {
		return false
	}

	if regErr, ok := err.(*registryError); ok {
		return regErr.Reason == reason
	}
	return false
}
