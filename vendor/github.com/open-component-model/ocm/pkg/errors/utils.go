// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"errors"
	"syscall"
)

// IsRetryable checks whether a retry should be performed for a failed operation.
func IsRetryable(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
