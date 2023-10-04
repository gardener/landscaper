// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociblob

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/options"
)

func ConfigHandler() flagsets.ConfigOptionTypeSetHandler {
	return flagsets.NewConfigOptionTypeSetHandler(
		Type, AddConfig,
		options.ReferenceOption,
		options.MediatypeOption,
		options.SizeOption,
		options.DigestOption,
	)
}

func AddConfig(opts flagsets.ConfigOptions, config flagsets.Config) error {
	flagsets.AddFieldByOptionP(opts, options.ReferenceOption, config, "ref")
	flagsets.AddFieldByOptionP(opts, options.MediatypeOption, config, "mediaType")
	flagsets.AddFieldByOptionP(opts, options.SizeOption, config, "size")
	flagsets.AddFieldByOptionP(opts, options.DigestOption, config, "digest")
	return nil
}

var usage = `
This method implements the access of an OCI blob stored in an OCI repository.
`

var formatV1 = `
The type specific specification fields are:

- **<code>imageReference</code>** *string*

  OCI repository reference (this artifact name used to store the blob).

- **<code>mediaType</code>** *string*

  The media type of the blob

- **<code>digest</code>** *string*

  The digest of the blob used to access the blob in the OCI repository.

- **<code>size</code>** *integer*

  The size of the blob
`
