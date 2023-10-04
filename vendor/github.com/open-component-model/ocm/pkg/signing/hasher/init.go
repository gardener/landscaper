// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hasher

import (
	_ "github.com/open-component-model/ocm/pkg/signing/hasher/nodigest"
	_ "github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
	_ "github.com/open-component-model/ocm/pkg/signing/hasher/sha512"
)
