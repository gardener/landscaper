// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

type LsError interface {
	error
	LandscaperError() *lsv1alpha1.Error
	Unwrap() error
	UpdatedError(lastError *lsv1alpha1.Error) *lsv1alpha1.Error
}

// Error is a wrapper around the landscaper crd error
// that implements the go error interface.
type Error struct {
	lsErr lsv1alpha1.Error
	err   error
}

// Error implements the error interface
func (e Error) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("Op: %s - Reason: %s - Message: %s", e.lsErr.Operation, e.lsErr.Reason, e.lsErr.Message)
}

// LandscaperError returns the wrapped landscaper error.
func (e Error) LandscaperError() *lsv1alpha1.Error {
	return e.lsErr.DeepCopy()
}

// Unwrap implements the unwrap interface
func (e Error) Unwrap() error {
	return e.err
}

// UpdatedError updates the properties of an existing error.
func (e Error) UpdatedError(lastError *lsv1alpha1.Error) *lsv1alpha1.Error {
	return UpdatedError(lastError, e.lsErr.Operation, e.lsErr.Reason, e.lsErr.Message, e.lsErr.Codes...)
}

// NewError creates a new landscaper internal error
func NewError(operation, reason, message string, codes ...lsv1alpha1.ErrorCode) *Error {
	return &Error{
		lsErr: lsv1alpha1.Error{
			Operation:          operation,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Codes:              codes,
		},
	}
}

// NewWrappedError creates a new landscaper internal error that wraps another error
func NewWrappedError(err error, operation, reason, message string, codes ...lsv1alpha1.ErrorCode) LsError {
	return &Error{
		lsErr: lsv1alpha1.Error{
			Operation:          operation,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Codes:              codes,
		},
		err: err,
	}
}

// NewErrorOrNil creates a new landscaper internal error that wraps another error.
// if no error is set the functions return nil.
// The error is automatically set as error message.
func NewErrorOrNil(err error, operation, reason string, codes ...lsv1alpha1.ErrorCode) LsError {
	if err == nil {
		return nil
	}
	return &Error{
		lsErr: lsv1alpha1.Error{
			Operation:          operation,
			Reason:             reason,
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Codes:              codes,
		},
		err: err,
	}
}

// BuildLsError creates a new landscaper internal error if the provided error is not already of such a type or nil.
// Otherwise the error is returned.
func BuildLsError(err error, operation, reason, message string, codes ...lsv1alpha1.ErrorCode) LsError {
	if err == nil {
		return NewWrappedError(err, operation, reason, message, codes...)
	}

	switch e := err.(type) {
	case LsError:
		return e
	default:
		return NewWrappedError(err, operation, reason, message, codes...)
	}
}

// BuildLsErrorOrNil creates a new landscaper internal error if the provided error is not already of such a type or nil.
// Otherwise the error is returned. If the input error is nil also nil is returned.
func BuildLsErrorOrNil(err error, operation, reason string, codes ...lsv1alpha1.ErrorCode) LsError {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case LsError:
		return e
	default:
		return NewErrorOrNil(err, operation, reason, codes...)
	}
}

// IsError returns the landscaper error if the given error is one.
// If the err does not contain a landscaper error nil is returned.
func IsError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	switch e := err.(type) {
	case *Error:
		return e, true
	default:
		uErr := errors.Unwrap(err)
		if uErr == nil {
			return nil, false
		}
		return IsError(uErr)
	}
}

// TryUpdateError tries to update the properties of the last error if the err is a internal landscaper error.
func TryUpdateError(lastErr *lsv1alpha1.Error, err error) *lsv1alpha1.Error {
	if err == nil {
		return nil
	}
	if intErr, ok := IsError(err); ok {
		return intErr.UpdatedError(lastErr)
	}
	return nil
}

// TryUpdateLsError tries to update the properties of the last error if the err is a internal landscaper error.
func TryUpdateLsError(lastErr *lsv1alpha1.Error, err LsError) *lsv1alpha1.Error {
	if err == nil {
		return nil
	}

	codes := CollectErrorCodes(err)

	errorInfo := err.LandscaperError()
	return UpdatedError(lastErr, errorInfo.Operation, errorInfo.Reason, errorInfo.Message, codes...)
}

func CollectErrorCodes(err error) []lsv1alpha1.ErrorCode {
	codes := []lsv1alpha1.ErrorCode{}
	subError := errors.Unwrap(err)
	if subError != nil {
		codes = CollectErrorCodes(subError)
	}

	switch e := err.(type) {
	case LsError:
		codes = append(codes, e.LandscaperError().Codes...)
	default:
		// nothing
	}

	return codes
}

// UpdatedError updates the properties of a error.
func UpdatedError(lastError *lsv1alpha1.Error, operation, reason, message string, codes ...lsv1alpha1.ErrorCode) *lsv1alpha1.Error {
	if lastError == nil {
		return &lsv1alpha1.Error{
			Operation:          operation,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Now(),
			LastUpdateTime:     metav1.Now(),
			Codes:              codes,
		}
	}

	newError := &lsv1alpha1.Error{
		Operation:          operation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: lastError.LastTransitionTime,
		LastUpdateTime:     lastError.LastUpdateTime,
		Codes:              codes,
	}

	// Normalize nil and empty slice
	if len(lastError.Codes) == 0 && len(codes) == 0 {
		newError.Codes = lastError.Codes
	}

	if !reflect.DeepEqual(lastError, newError){
		newError.LastUpdateTime = metav1.Now()
	}

	if lastError.Operation != operation {
		newError.LastTransitionTime = metav1.Now()
	}

	return newError
}

// GetPhaseForLastError returns a failed installation phase if the given
// error lasts longer than the specified time.
func GetPhaseForLastError(phase lsv1alpha1.ComponentInstallationPhase, lastError *lsv1alpha1.Error, d time.Duration) lsv1alpha1.ComponentInstallationPhase {
	if lastError == nil {
		return phase
	}
	if len(phase) == 0 {
		return lsv1alpha1.ComponentPhaseFailed
	}

	// directly set the phase to error if the error contains an unrecoverable error code
	if ContainsAnyErrorCode(lastError.Codes, lsv1alpha1.UnrecoverableErrorCodes) {
		return lsv1alpha1.ComponentPhaseFailed
	}

	if lastError.LastUpdateTime.Sub(lastError.LastTransitionTime.Time).Seconds() > d.Seconds() {
		return lsv1alpha1.ComponentPhaseFailed
	}
	return phase
}

// ContainsAnyErrorCode checks whether any expected error code is included in a list of error codes
func ContainsAnyErrorCode(codes []lsv1alpha1.ErrorCode, expected []lsv1alpha1.ErrorCode) bool {
	for _, expected := range expected {
		if HasErrorCode(codes, expected) {
			return true
		}
	}
	return false
}

// HasErrorCode checks if a code is present in the a list of error codes.
func HasErrorCode(codes []lsv1alpha1.ErrorCode, expected lsv1alpha1.ErrorCode) bool {
	for _, code := range codes {
		if code == expected {
			return true
		}
	}
	return false
}
