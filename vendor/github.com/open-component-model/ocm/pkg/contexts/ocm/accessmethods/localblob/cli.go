// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package localblob

import (
	"github.com/open-component-model/ocm/pkg/cobrautils/flagsets"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/options"
)

func ConfigHandler() flagsets.ConfigOptionTypeSetHandler {
	return flagsets.NewConfigOptionTypeSetHandler(
		Type, AddConfig,
		options.ReferenceOption,
		options.MediatypeOption,
		options.HintOption,
		options.GlobalAccessOption,
	)
}

func AddConfig(opts flagsets.ConfigOptions, config flagsets.Config) error {
	flagsets.AddFieldByOptionP(opts, options.ReferenceOption, config, "localReference")
	flagsets.AddFieldByOptionP(opts, options.HintOption, config, "referenceName")
	flagsets.AddFieldByOptionP(opts, options.MediatypeOption, config, "mediaType")
	flagsets.AddFieldByOptionP(opts, options.GlobalAccessOption, config, "globalAccess")
	return nil
}

var usage = `
This method is used to store a resource blob along with the component descriptor
on behalf of the hosting OCM repository.

Its implementation is specific to the implementation of OCM
repository used to read the component descriptor. Every repository
implementation may decide how and where local blobs are stored,
but it MUST provide an implementation for this method.

Regardless of the chosen implementation the attribute specification is
defined globally the same.
`

var formatV1 = `
The type specific specification fields are:

- **<code>localReference</code>** *string*

  Repository type specific location information as string. The value
  may encode any deep structure, but typically just an access path is sufficient.

- **<code>mediaType</code>** *string*

  The media type of the blob used to store the resource. It may add
  format information like <code>+tar</code> or <code>+gzip</code>.

- **<code>referenceName</code>** (optional) *string*

  This optional attribute may contain identity information used by
  other repositories to restore some global access with an identity
  related to the original source.

  For example, if an OCI artifact originally referenced using the
  access method <code>ociArtifact</code> is stored during
  some transport step as local artifact, the reference name can be set
  to its original repository name. An import step into an OCI based OCM
  repository may then decide to make this artifact available again as
  regular OCI artifact.

- **<code>globalAccess</code>** (optional) *access method specification*

  If a resource blob is stored locally, the repository implementation
  may decide to provide an external access information (independent
  of the OCM model).

  For example, an OCI artifact stored as local blob
  can be additionally stored as regular OCI artifact in an OCI registry.

  This additional external access information can be added using
  a second external access method specification.
`
