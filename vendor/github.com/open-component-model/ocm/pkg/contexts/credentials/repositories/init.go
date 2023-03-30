// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package repositories

import (
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/aliases"
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/directcreds"
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig"
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/memory"
	_ "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/memory/config"
)
