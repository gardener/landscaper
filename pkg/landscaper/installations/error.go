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

package installations

import "fmt"

// ErrorReason describes specific import error reasons
type ErrorReason string

const (
	ImportNotFound     ErrorReason = "ImportNotFound"
	ImportNotSatisfied ErrorReason = "ImportNotSatisfied"
	ExportNotFound     ErrorReason = "ExportNotFound"
)

type Error struct {
	Reason  ErrorReason
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s - %s", e.Reason, e.Message)
	}
	return fmt.Sprintf("%s - %s: %s", e.Reason, e.Message, e.Err.Error())
}

// Unwrap implements the golang Unwrap function
func (e *Error) Unwrap() error { return e.Err }

// NewError creates a new import error
func NewError(reason ErrorReason, message string, err error) error {
	return &Error{
		Reason:  reason,
		Message: message,
		Err:     err,
	}
}

// NewErrorf creates a new import error with a formated message
func NewErrorf(reason ErrorReason, err error, format string, a ...interface{}) error {
	return &Error{
		Reason:  reason,
		Message: fmt.Sprintf(format, a...),
		Err:     err,
	}
}

// NewImportNotFoundError creates a new error that indicates that a import was not found
func NewImportNotFoundError(message string, err error) error {
	return NewError(ImportNotFound, message, err)
}

// NewImportNotFoundErrorf creates a new error that indicates that a import was not found with a formatted message
func NewImportNotFoundErrorf(err error, format string, a ...interface{}) error {
	return NewErrorf(ImportNotFound, err, format, a...)
}

// IsImportNotFoundError checks if the provided error is of type ImportNotFound
func IsImportNotFoundError(err error) bool {
	return IsErrorForReason(err, ImportNotFound)
}

// NewImportNotFoundError creates a new error that indicates that a import is not satisfied yet
func NewImportNotSatisfiedError(message string, err error) error {
	return NewError(ImportNotSatisfied, message, err)
}

// NewImportNotSatisfiedErrorf creates a new error that indicates that a import was not found with a formatted message
func NewImportNotSatisfiedErrorf(err error, format string, a ...interface{}) error {
	return NewErrorf(ImportNotSatisfied, err, format, a...)
}

// IsImportNotFoundError checks if the provided error is of type ImportNotSatisfied
func IsImportNotSatisfiedError(err error) bool {
	return IsErrorForReason(err, ImportNotSatisfied)
}

// NewExportNotFoundError creates a new error that indicates that a export was not found
func NewExportNotFoundError(message string, err error) error {
	return NewError(ExportNotFound, message, err)
}

// NewExportNotFoundErrorf creates a new error that indicates that a import was not found with a formatted message
func NewExportNotFoundErrorf(err error, format string, a ...interface{}) error {
	return NewErrorf(ExportNotFound, err, format, a...)
}

// IsExportNotFoundError checks if the provided error is of type ExportNotFound
func IsExportNotFoundError(err error) bool {
	return IsErrorForReason(err, ExportNotFound)
}

// IsErrorForReason checks if the error is a registry error and of the givne reason.
func IsErrorForReason(err error, reason ErrorReason) bool {
	if err == nil {
		return false
	}

	if regErr, ok := err.(*Error); ok {
		return regErr.Reason == reason
	}
	return false
}
