// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package npm

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/options"
)

func ConfigHandler() flagsets.ConfigOptionTypeSetHandler {
	return flagsets.NewConfigOptionTypeSetHandler(
		Type, AddConfig,
		options.RegistryOption,
		options.PackageOption,
		options.VersionOption,
	)
}

func AddConfig(opts flagsets.ConfigOptions, config flagsets.Config) error {
	flagsets.AddFieldByOptionP(opts, options.RegistryOption, config, "registry")
	flagsets.AddFieldByOptionP(opts, options.PackageOption, config, "package")
	flagsets.AddFieldByOptionP(opts, options.VersionOption, config, "version")
	return nil
}

var usage = `
This method implements the access of an NPM package in an NPM registry.
`

var formatV1 = `
The type specific specification fields are:

- **<code>registry</code>** *string*

  Base URL of the NPM registry.

- **<code>package</code>** *string*

  The name of the NPM package

- **<code>version</code>** *string*

  The version name of the NPM package
`
