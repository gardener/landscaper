// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	_ "github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs"
	_ "github.com/open-component-model/ocm/pkg/contexts/oci"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/config"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/blobhandler/handlers"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/normalizations"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/versions"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/config"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/digester/digesters"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/download/config"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/download/handlers"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/repositories"
	_ "github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/handlers"
)
