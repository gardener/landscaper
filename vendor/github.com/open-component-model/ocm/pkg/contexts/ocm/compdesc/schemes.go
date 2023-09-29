// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"sort"

	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
)

const DefaultSchemeVersion = "v2"

type ComponentDescriptorVersion interface {
	SchemaVersion() string
	GetName() string
	GetVersion() string
	Normalize(normAlgo string) ([]byte, error)
}

type Scheme interface {
	GetVersion() string

	Decode(data []byte, opts *DecodeOptions) (ComponentDescriptorVersion, error)
	ConvertFrom(desc *ComponentDescriptor) (ComponentDescriptorVersion, error)
	ConvertTo(ComponentDescriptorVersion) (*ComponentDescriptor, error)
}

type Schemes map[string]Scheme

func (v Schemes) Register(scheme Scheme) {
	v[scheme.GetVersion()] = scheme
}

func (v Schemes) Names() []string {
	names := []string{}
	for n := range v {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

var DefaultSchemes = Schemes{}

func RegisterScheme(scheme Scheme) {
	DefaultSchemes.Register(scheme)
}

////////////////////////////////////////////////////////////////////////////////

// Decode decodes a component into the given object.
func Decode(data []byte, opts ...DecodeOption) (*ComponentDescriptor, error) {
	o := &DecodeOptions{Codec: DefaultYAMLCodec}
	o.ApplyOptions(opts)

	var schemedef struct {
		Meta       metav1.Metadata `json:"meta"`
		APIVersion string          `json:"apiVersion"`
	}
	if err := o.Codec.Decode(data, &schemedef); err != nil {
		Logger.Debug("decoding of component descriptor failed", "error", err.Error(), "data", string(data))
		return nil, err
	}

	scheme := schemedef.Meta.Version
	if schemedef.APIVersion != "" {
		if scheme != "" {
			return nil, errors.Newf("apiVersion and meta.schemeVersion defined")
		}
		scheme = schemedef.APIVersion
	}
	version := DefaultSchemes[scheme]
	if version == nil {
		return nil, errors.ErrNotSupported(errors.KIND_SCHEMAVERSION, scheme)
	}

	versioned, err := version.Decode(data, o)
	if err != nil {
		Logger.Debug("versioned decoding of component descriptor failed", "error", err.Error(), "scheme", scheme, "data", string(data))
		return nil, err
	}
	cd, err := version.ConvertTo(versioned)
	if err != nil {
		Logger.Debug("conversion of component descriptor failed", "error", err, "scheme", scheme, "data", string(data))
	}
	return cd, err
}

// DecodeOptions defines decode options for the codec.
type DecodeOptions struct {
	Codec             Codec
	DisableValidation bool
	StrictMode        bool
}

var _ DecodeOption = &DecodeOptions{}

// ApplyDecodeOption applies the actual options.
func (o *DecodeOptions) ApplyDecodeOption(options *DecodeOptions) {
	if o == nil {
		return
	}
	if o.Codec != nil {
		options.Codec = o.Codec
	}
	options.DisableValidation = o.DisableValidation
	options.StrictMode = o.StrictMode
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *DecodeOptions) ApplyOptions(opts []DecodeOption) *DecodeOptions {
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyDecodeOption(o)
		}
	}
	return o
}

// DecodeOption is the interface to specify different cache options.
type DecodeOption interface {
	ApplyDecodeOption(options *DecodeOptions)
}

// StrictMode enables or disables strict mode parsing.
type StrictMode bool

// ApplyDecodeOption applies the configured strict mode.
func (s StrictMode) ApplyDecodeOption(options *DecodeOptions) {
	options.StrictMode = bool(s)
}

// DisableValidation enables or disables validation of the component descriptor.
type DisableValidation bool

// ApplyDecodeOption applies the validation disable option.
func (v DisableValidation) ApplyDecodeOption(options *DecodeOptions) {
	options.DisableValidation = bool(v)
}

////////////////////////////////////////////////////////////////////////////////

// Encode encodes a component into the given object.
// If the serialization version is left blank, the schema version configured in the
// component descriptor will be used.
func Encode(obj *ComponentDescriptor, opts ...EncodeOption) ([]byte, error) {
	o := (&EncodeOptions{}).ApplyOptions(opts).DefaultFor(obj)
	v, err := Convert(obj, o)
	if err != nil {
		return nil, err
	}
	return o.Codec.Encode(v)
}

// Convert converts a component descriptor into a dedicated scheme version.
// If the serialization version is left blank, the schema version configured in the
// component descriptor will be used.
func Convert(obj *ComponentDescriptor, opts ...EncodeOption) (ComponentDescriptorVersion, error) {
	o := (&EncodeOptions{}).ApplyOptions(opts).DefaultFor(obj)
	cv := DefaultSchemes[o.SchemaVersion]
	if cv == nil {
		if cv == nil {
			return nil, errors.ErrNotSupported(errors.KIND_SCHEMAVERSION, o.SchemaVersion)
		}
	}
	return cv.ConvertFrom(obj)
}

////////////////////////////////////////////////////////////////////////////////

type EncodeOptions struct {
	Codec         Codec
	SchemaVersion string
}

var _ EncodeOption = &EncodeOptions{}

// ApplyDecodeOption applies the actual options.
func (o *EncodeOptions) ApplyEncodeOption(options *EncodeOptions) {
	if o == nil {
		return
	}
	if o.Codec != nil {
		options.Codec = o.Codec
	}
	if o.SchemaVersion != "" {
		options.SchemaVersion = o.SchemaVersion
	}
}

func (o *EncodeOptions) DefaultFor(cd *ComponentDescriptor) *EncodeOptions {
	if o.Codec == nil {
		o.Codec = DefaultYAMLCodec
	}
	if o.SchemaVersion == "" {
		o.SchemaVersion = cd.Metadata.ConfiguredVersion
	}
	if o.SchemaVersion == "" {
		o.SchemaVersion = DefaultSchemeVersion
	}
	return o
}

// ApplyOptions applies the given list options on these options,
// and then returns itself (for convenient chaining).
func (o *EncodeOptions) ApplyOptions(opts []EncodeOption) *EncodeOptions {
	for _, opt := range opts {
		if opt != nil {
			opt.ApplyEncodeOption(o)
		}
	}
	return o
}

// EncodeOption is the interface to specify different encode options.
type EncodeOption interface {
	ApplyEncodeOption(options *EncodeOptions)
}

// SchemaVersion enforces a dedicated schema version .
type SchemaVersion string

// ApplyEncodeOption applies the configured schema version.
func (o SchemaVersion) ApplyEncodeOption(options *EncodeOptions) {
	options.SchemaVersion = string(o)
}

// CodecWrappers can be used as EncodeOption, also
