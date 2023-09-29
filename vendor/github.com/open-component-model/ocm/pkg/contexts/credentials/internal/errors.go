// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	KIND_CREDENTIALS = "credentials"
	KIND_CONSUMER    = "consumer"
	KIND_REPOSITORY  = "repository"
)

func ErrUnknownCredentials(name string) error {
	return errors.ErrUnknown(KIND_CREDENTIALS, name)
}

func ErrUnknownConsumer(name string) error {
	return errors.ErrUnknown(KIND_CONSUMER, name)
}

func ErrUnknownRepository(kind, name string) error {
	return errors.ErrUnknown(KIND_REPOSITORY, name, kind)
}
