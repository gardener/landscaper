// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dirtree

import (
	"fmt"

	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/download"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/listformat"
	"github.com/open-component-model/ocm/pkg/registrations"
)

func init() {
	download.RegisterHandlerRegistrationHandler("ocm/dirtree", &RegistrationHandler{})
}

type Config struct {
	AsArchive      bool     `json:"asArchive"`
	OCIConfigTypes []string `json:"ociConfigTypes"`
}

func AttributeDescription() map[string]string {
	return map[string]string{
		"asArchive": "flag to request an archive download",
		"ociConfigTypes": "a list of accepted OCI config archive mime types\n" +
			"defaulted by <code>" + ociv1.MediaTypeImageConfig + "</code>.",
	}
}

type RegistrationHandler struct{}

var _ download.HandlerRegistrationHandler = (*RegistrationHandler)(nil)

func (r *RegistrationHandler) RegisterByName(handler string, ctx download.Target, config download.HandlerConfig, olist ...download.HandlerOption) (bool, error) {
	var err error

	if handler != "" {
		return true, fmt.Errorf("invalid dirtree handler %q", handler)
	}

	attr, err := registrations.DecodeDefaultedConfig[Config](config)
	if err != nil {
		return true, errors.Wrapf(err, "cannot unmarshal download handler configuration")
	}

	opts := download.NewHandlerOptions(olist...)
	if opts.MimeType != "" && !slices.Contains(supportedMimeTypes, opts.MimeType) {
		return true, errors.Wrapf(err, "mime type %s not supported", opts.MimeType)
	}
	if opts.ArtifactType != "" && slices.Contains(defaultArtifactTypes, opts.ArtifactType) && !attr.AsArchive {
		return true, nil
	}

	h := New(attr.OCIConfigTypes...).SetArchiveMode(attr.AsArchive)
	if opts.MimeType == "" {
		for _, m := range supportedMimeTypes {
			opts.MimeType = m
			download.For(ctx).Register(h, opts)
		}
	} else {
		download.For(ctx).Register(h, opts)
	}

	return true, nil
}

func (r *RegistrationHandler) GetHandlers(ctx cpi.Context) registrations.HandlerInfos {
	return registrations.NewLeafHandlerInfo("downloading directory tree-like resources", `
The <code>dirtree</code> downloader is able to download directory-tree like
resources as directory structure (default) or archive.
The following artifact media types are supported:
`+listformat.FormatList("", SupportedMimeTypes()...)+`
By default, it is registered for the following resource types:
`+listformat.FormatList("", defaultArtifactTypes...)+`
It accepts a config with the following fields:
`+listformat.FormatMapElements("", AttributeDescription()),
	)
}
