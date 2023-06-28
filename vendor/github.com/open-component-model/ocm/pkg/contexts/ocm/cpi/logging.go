// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"github.com/mandelsoft/logging"
)

type OCMContextProvider interface {
	GetContext() Context
}

func Logger(c OCMContextProvider, keyValuePairs ...interface{}) logging.Logger {
	return c.GetContext().Logger().WithValues(keyValuePairs...)
}
