// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"fmt"

	lserror "github.com/gardener/landscaper/apis/errors"
)

// ErrorReason describes specific import error reasons
type ErrorReason string

const (
	ImportNotFound         ErrorReason = "ImportNotFound"
	ImportNotSatisfied     ErrorReason = "ImportNotSatisfied"
	InvalidDefaultValue    ErrorReason = "InvalidDefaultValue"
	NotCompletedDependents ErrorReason = "NotCompletedDependents"
	SchemaValidationFailed ErrorReason = "SchemaValidationFailed"
)

// NewErrorf creates a new import error with a formated message
func NewErrorf(reason ErrorReason, err error, format string, a ...interface{}) lserror.LsError {
	return lserror.NewWrappedError(err, string(reason), string(reason), fmt.Sprintf(format, a...))
}

// NewImportNotFoundErrorf creates a new error that indicates that a import was not found with a formatted message
func NewImportNotFoundErrorf(err error, format string, a ...interface{}) lserror.LsError {
	return NewErrorf(ImportNotFound, err, format, a...)
}

// NewImportNotSatisfiedErrorf creates a new error that indicates that a import was not found with a formatted message
func NewImportNotSatisfiedErrorf(err error, format string, a ...interface{}) lserror.LsError {
	return NewErrorf(ImportNotSatisfied, err, format, a...)
}

// NewNotCompletedDependentsErrorf creates a new error that indicates that dependent installation is not completed yet
func NewNotCompletedDependentsErrorf(err error, format string, a ...interface{}) lserror.LsError {
	return NewErrorf(NotCompletedDependents, err, format, a...)
}

// IsNotCompletedDependentsError checks if the provided error is of type NotCompletedDependents
func IsNotCompletedDependentsError(err error) bool {
	return IsErrorForReason(err, NotCompletedDependents)
}

// IsSchemaValidationFailedError checks if the provided error is of type SchemaValidationFailed
func IsSchemaValidationFailedError(err error) bool {
	return IsErrorForReason(err, SchemaValidationFailed)
}

func IsImportNotFoundError(err error) bool {
	return IsErrorForReason(err, ImportNotFound)
}

// IsErrorForReason checks if the error is a registry error and of the givne reason.
func IsErrorForReason(err error, reason ErrorReason) bool {
	if err == nil {
		return false
	}

	if lsErr, ok := err.(lserror.LsError); ok {
		return lsErr.LandscaperError().Reason == string(reason)
	}

	return false
}
