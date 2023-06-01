// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Options struct {
	accessio.StandardOptions

	FormatVersion string `json:"formatVersion,omitempty"`
}

func NewOptions(olist ...accessio.Option) (*Options, error) {
	opts := &Options{}
	err := accessio.ApplyOptions(opts, olist...)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

type FormatVersionOption interface {
	SetFormatVersion(string)
	GetFormatVersion() string
}

func GetFormatVersion(opts accessio.Options) string {
	if o, ok := opts.(FormatVersionOption); ok {
		return o.GetFormatVersion()
	}
	return ""
}

var _ FormatVersionOption = (*Options)(nil)

func (o *Options) SetFormatVersion(s string) {
	o.FormatVersion = s
}

func (o *Options) GetFormatVersion() string {
	return o.FormatVersion
}

func (o *Options) ApplyOption(opts accessio.Options) error {
	err := o.StandardOptions.ApplyOption(opts)
	if err != nil {
		return err
	}
	if o.FormatVersion != "" {
		if s, ok := opts.(FormatVersionOption); ok {
			s.SetFormatVersion(o.FormatVersion)
		} else {
			return errors.ErrNotSupported("format version option")
		}
	}
	return nil
}

type optFmt struct {
	format string
}

var _ accessio.Option = (*optFmt)(nil)

func StructureFormat(fmt string) accessio.Option {
	return &optFmt{fmt}
}

func (o *optFmt) ApplyOption(opts accessio.Options) error {
	if s, ok := opts.(FormatVersionOption); ok {
		s.SetFormatVersion(o.format)
		return nil
	}
	return errors.ErrNotSupported("format version option")
}
