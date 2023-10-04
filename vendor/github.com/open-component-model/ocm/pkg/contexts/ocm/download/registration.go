// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/registrations"
)

type Target = cpi.Context

////////////////////////////////////////////////////////////////////////////////

type HandlerOptions struct {
	HandlerKey `json:",inline"`
	Priority   int `json:"priority,omitempty"`
}

func NewHandlerOptions(olist ...HandlerOption) *HandlerOptions {
	var opts HandlerOptions
	for _, o := range olist {
		o.ApplyHandlerOptionTo(&opts)
	}
	return &opts
}

func (o *HandlerOptions) ApplyHandlerOptionTo(opts *HandlerOptions) {
	if o.Priority > 0 {
		opts.Priority = o.Priority
	}
	o.HandlerKey.ApplyHandlerOptionTo(opts)
}

type HandlerOption interface {
	ApplyHandlerOptionTo(*HandlerOptions)
}

////////////////////////////////////////////////////////////////////////////////

// HandlerKey is the registration key for download handlers.
type HandlerKey struct {
	ArtifactType string `json:"artifactType,omitempty"`
	MimeType     string `json:"mimeType,omitempty"`
}

var _ HandlerOption = HandlerKey{}

func NewHandlerKey(artifactType, mimetype string) HandlerKey {
	return HandlerKey{
		ArtifactType: artifactType,
		MimeType:     mimetype,
	}
}

func (k HandlerKey) ApplyHandlerOptionTo(opts *HandlerOptions) {
	if k.ArtifactType != "" {
		opts.ArtifactType = k.ArtifactType
	}
	if k.MimeType != "" {
		opts.MimeType = k.MimeType
	}
}

func ForCombi(artifacttype string, mimetype string) HandlerOption {
	return HandlerKey{ArtifactType: artifacttype, MimeType: mimetype}
}

func ForMimeType(mimetype string) HandlerOption {
	return HandlerKey{MimeType: mimetype}
}

func ForArtifactType(artifacttype string) HandlerOption {
	return HandlerKey{ArtifactType: artifacttype}
}

type prio struct {
	prio int
}

func WithPrio(p int) HandlerOption {
	return prio{p}
}

func (o prio) ApplyHandlerOptionTo(opts *HandlerOptions) {
	opts.Priority = o.prio
}

////////////////////////////////////////////////////////////////////////////////

type (
	HandlerConfig               = registrations.HandlerConfig
	HandlerRegistrationHandler  = registrations.HandlerRegistrationHandler[Target, HandlerOption]
	HandlerRegistrationRegistry = registrations.HandlerRegistrationRegistry[Target, HandlerOption]

	RegistrationHandlerInfo = registrations.RegistrationHandlerInfo[Target, HandlerOption]
)

func NewHandlerRegistrationRegistry(base ...HandlerRegistrationRegistry) HandlerRegistrationRegistry {
	return registrations.NewHandlerRegistrationRegistry[Target, HandlerOption](base...)
}

func NewRegistrationHandlerInfo(path string, handler HandlerRegistrationHandler) *RegistrationHandlerInfo {
	return registrations.NewRegistrationHandlerInfo[Target, HandlerOption](path, handler)
}

func RegisterHandlerRegistrationHandler(path string, handler HandlerRegistrationHandler) {
	DefaultRegistry.RegisterRegistrationHandler(path, handler)
}

func RegisterHandlerByName(ctx cpi.ContextProvider, name string, config HandlerConfig, opts ...HandlerOption) error {
	hdlrs := For(ctx)
	o, err := hdlrs.RegisterByName(name, ctx.OCMContext(), config, opts...)
	if err != nil {
		return err
	}
	if !o {
		return fmt.Errorf("no matching handler found for %q", name)
	}
	return nil
}
