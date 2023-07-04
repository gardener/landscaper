// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	_ "github.com/open-component-model/ocm/pkg/signing/handlers/rsa"
	_ "github.com/open-component-model/ocm/pkg/signing/handlers/rsa-signingservice"
	_ "github.com/open-component-model/ocm/pkg/signing/handlers/sigstore"
	_ "github.com/sigstore/cosign/v2/pkg/providers/all"
)
