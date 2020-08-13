// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"fmt"
)

// ErrorReason represents a registry specific issue
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

// NewComponentNotFoundError creates a new ComponentNotFoundError
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

// IsErrorForReason checks if the error is a registry error and of the givne reason.
func IsErrorForReason(err error, reason ErrorReason) bool {
	if err == nil {
		return false
	}

	if regErr, ok := err.(*registryError); ok {
		return regErr.Reason == reason
	}
	return false
}
