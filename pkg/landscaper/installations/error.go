// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import "fmt"

// ErrorReason describes specific import error reasons
type ErrorReason string

const (
	ImportNotFound         ErrorReason = "ImportNotFound"
	ImportNotSatisfied     ErrorReason = "ImportNotSatisfied"
	NotCompletedDependents ErrorReason = "NotCompletedDependents"
	ExportNotFound         ErrorReason = "ExportNotFound"
	SchemaValidationFailed ErrorReason = "SchemaValidationFailed"
)

type Error struct {
	Reason  ErrorReason
	Message string
	Err     error
}

func (e *Error) Error() string {
	msg := string(e.Reason)
	if len(e.Message) != 0 {
		msg = fmt.Sprintf("%s - %s", e.Reason, e.Message)
	}
	if e.Err != nil {
		msg = fmt.Sprintf("%s: ", e.Err.Error())
	}
	return msg
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

// NewError creates a new import error
func NewErrorWrap(reason ErrorReason, err error) error {
	return &Error{
		Reason: reason,
		Err:    err,
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

// NewNotCompletedDependentsError creates a new error that indicates that dependent installation is not completed yet
func NewNotCompletedDependentsError(message string, err error) error {
	return NewError(NotCompletedDependents, message, err)
}

// NewNotCompletedDependentsErrorf creates a new error that indicates that dependent installation is not completed yet
func NewNotCompletedDependentsErrorf(err error, format string, a ...interface{}) error {
	return NewErrorf(NotCompletedDependents, err, format, a...)
}

// IsNotCompletedDependentsError checks if the provided error is of type NotCompletedDependents
func IsNotCompletedDependentsError(err error) bool {
	return IsErrorForReason(err, NotCompletedDependents)
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

// IsSchemaValidationFailedError checks if the provided error is of type SchemaValidationFailed
func IsSchemaValidationFailedError(err error) bool {
	return IsErrorForReason(err, SchemaValidationFailed)
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
