// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/errors"
)

const KIND_CONFIGTYPE = "config type"

// OCM_CONFIG_SUFFIX is the standard suffix used for all configuration types
// provided by this library.
const OCM_CONFIG_SUFFIX = ".config.ocm.software"

////////////////////////////////////////////////////////////////////////////////

type noContextError struct {
	name string
}

func (e *noContextError) Error() string {
	return fmt.Sprintf("unknown context %q", e.name)
}

func ErrNoContext(name string) error {
	return &noContextError{name}
}

func IsErrNoContext(err error) bool {
	return errors.IsA(err, &noContextError{})
}

func IsErrConfigNotApplicable(err error) bool {
	return errors.IsErrUnknownKind(err, KIND_CONFIGTYPE) || IsErrNoContext(err)
}
