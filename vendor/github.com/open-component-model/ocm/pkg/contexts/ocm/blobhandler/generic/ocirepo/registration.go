// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocirepo

import (
	"encoding/json"
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/ociuploadattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type Config = ociuploadattr.Attribute

func init() {
	cpi.RegisterBlobHandlerRegistrationHandler("ocm/ociArtifacts", &RegistrationHandler{})
}

type RegistrationHandler struct{}

var _ cpi.BlobHandlerRegistrationHandler = (*RegistrationHandler)(nil)

func (r *RegistrationHandler) RegisterByName(handler string, ctx cpi.Context, config cpi.BlobHandlerConfig, olist ...cpi.BlobHandlerOption) (bool, error) {
	if handler != "" {
		return true, fmt.Errorf("invalid ociArtifact handler %q", handler)
	}
	if config == nil {
		return true, fmt.Errorf("oci target specification required")
	}
	var attr *Config
	var ok bool
	switch a := config.(type) {
	case *ociuploadattr.Attribute:
		attr = a
	case json.RawMessage:
		r, err := ociuploadattr.AttributeType{}.Decode(a, runtime.DefaultYAMLEncoding)
		if err != nil {
			return true, errors.Wrapf(err, "cannot unmarshal blob handler target configuration")
		}
		attr, ok = r.(*ociuploadattr.Attribute)
		if !ok {
			return true, fmt.Errorf("failed to assert type %T to ociuploadattr.Attribute", r)
		}
	case []byte:
		r, err := ociuploadattr.AttributeType{}.Decode(a, runtime.DefaultYAMLEncoding)
		if err != nil {
			return true, errors.Wrapf(err, "cannot unmarshal blob handler target configuration")
		}
		attr, ok = r.(*ociuploadattr.Attribute)
		if !ok {
			return true, fmt.Errorf("failed to assert type %T to ociuploadattr.Attribute", r)
		}
	default:
		return true, fmt.Errorf("unexpected type %T for oci blob handler target", a)
	}

	var mimes []string
	opts := cpi.NewBlobHandlerOptions(olist...)
	if opts.MimeType != "" {
		found := false
		for _, a := range artdesc.ArchiveBlobTypes() {
			if a == opts.MimeType {
				found = true
				break
			}
		}
		if !found {
			return true, fmt.Errorf("unexpected type mime type %q for oci blob handler target", opts.MimeType)
		}
		mimes = append(mimes, opts.MimeType)
	} else {
		mimes = artdesc.ArchiveBlobTypes()
	}

	h := NewArtifactHandler(attr)
	for _, m := range mimes {
		opts.MimeType = m
		ctx.BlobHandlers().Register(h, opts)
	}

	return true, nil
}
