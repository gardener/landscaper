// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/open-component-model/ocm/pkg/signing"
)

// Hasher creates a new hash.Hash interface.
type Hasher = signing.Hasher

// HasherProvider provides access to supported hash methods.
type HasherProvider = signing.HasherProvider
