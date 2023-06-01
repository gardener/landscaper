// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/options"
)

func ConfigHandler() flagsets.ConfigOptionTypeSetHandler {
	return flagsets.NewConfigOptionTypeSetHandler(
		Type, AddConfig,
		options.RegionOption,
		options.BucketOption,
		options.ReferenceOption,
		options.MediatypeOption,
		options.VersionOption,
	)
}

func AddConfig(opts flagsets.ConfigOptions, config flagsets.Config) error {
	flagsets.AddFieldByOptionP(opts, options.ReferenceOption, config, "key")
	flagsets.AddFieldByOptionP(opts, options.MediatypeOption, config, "mediaType")
	flagsets.AddFieldByOptionP(opts, options.RegionOption, config, "region")
	flagsets.AddFieldByOptionP(opts, options.BucketOption, config, "bucket")
	flagsets.AddFieldByOptionP(opts, options.VersionOption, config, "version")
	return nil
}

var usage = `
This method implements the access of a blob stored in an S3 bucket.
`

var formatV1 = `
The type specific specification fields are:

- **<code>region</code>** (optional) *string*

  OCI repository reference (this artifact name used to store the blob).

- **<code>bucket</code>** *string*

  The name of the S3 bucket containing the blob

- **<code>key</code>** *string*

  The key of the desired blob
`
