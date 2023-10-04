// // SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
// //
// // SPDX-License-Identifier: Apache-2.0
package blueprint

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/registrations"
	"github.com/open-component-model/ocm/pkg/utils"
)

const PATH = "landscaper/blueprint"

func init() {
	download.RegisterHandlerRegistrationHandler(PATH, &RegistrationHandler{})
}

type Config struct {
	OCIConfigTypes []string `json:"ociConfigTypes"`
}

func AttributeDescription() map[string]string {
	return map[string]string{
		"ociConfigTypes": "a list of accepted OCI config archive mime types\n" +
			"defaulted by <code>" + CONFIG_MIME_TYPE + "</code>.",
	}
}

type RegistrationHandler struct{}

var _ download.HandlerRegistrationHandler = (*RegistrationHandler)(nil)

func (r *RegistrationHandler) RegisterByName(handler string, ctx download.Target, config download.HandlerConfig, olist ...download.HandlerOption) (bool, error) {
	var err error

	if handler != "" {
		return true, fmt.Errorf("invalid blueprint handler %q", handler)
	}

	opts := download.NewHandlerOptions(olist...)
	if opts.MimeType != "" && !slices.Contains(supportedArtifactTypes, opts.ArtifactType) {
		return false, errors.Newf("artifact type %s not supported", opts.ArtifactType)
	}

	if opts.MimeType != "" {
		if _, ok := mimeTypeExtractorRegistry[opts.MimeType]; !ok {
			return false, errors.Newf("mime type %s not supported", opts.MimeType)
		}
	}

	attr, err := registrations.DecodeDefaultedConfig[Config](config)
	if err != nil {
		return true, errors.Wrapf(err, "cannot unmarshal download handler configuration")
	}

	h := New(attr.OCIConfigTypes...)
	if opts.MimeType == "" {
		for m := range mimeTypeExtractorRegistry {
			opts.MimeType = m
			download.For(ctx).Register(h, opts)
		}
	} else {
		download.For(ctx).Register(h, opts)
	}

	return true, nil
}

func (r *RegistrationHandler) GetHandlers(ctx cpi.Context) registrations.HandlerInfos {
	return registrations.NewLeafHandlerInfo("uploading an OCI artifact to an OCI registry", `
The <code>artifact</code> downloader is able to transfer OCI artifact-like resources
into an OCI registry given by the combination of the download target and the
registration config.

If no config is given, the target must be an OCI reference with a potentially
omitted repository. The repo part is derived from the reference hint provided
by the resource's access specification.

If the config is given, the target is used as repository name prefixed with an
optional repository prefix given by the configuration.

The following artifact media types are supported:
`+listformat.FormatList("", utils.StringMapKeys(mimeTypeExtractorRegistry)...)+`
It accepts a config with the following fields:
`+listformat.FormatMapElements("", AttributeDescription())+`

This handler is by default registered for the following artifact types: 
`+strings.Join(supportedArtifactTypes, ","),
	)
}
