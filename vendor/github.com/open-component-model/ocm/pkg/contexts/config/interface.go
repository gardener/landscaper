// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/config/internal"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const KIND_CONFIGTYPE = internal.KIND_CONFIGTYPE

const OCM_CONFIG_TYPE_SUFFIX = internal.OCM_CONFIG_TYPE_SUFFIX

const CONTEXT_TYPE = internal.CONTEXT_TYPE

var AllConfigs = internal.AllConfigs

const AllGenerations = internal.AllGenerations

type (
	Context                = internal.Context
	ContextProvider        = internal.ContextProvider
	Config                 = internal.Config
	ConfigType             = internal.ConfigType
	ConfigTypeScheme       = internal.ConfigTypeScheme
	GenericConfig          = internal.GenericConfig
	ConfigSelector         = internal.ConfigSelector
	ConfigSelectorFunction = internal.ConfigSelectorFunction
)

func DefaultContext() internal.Context {
	return internal.DefaultContext
}

func ForContext(ctx context.Context) Context {
	return internal.ForContext(ctx)
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	return internal.DefinedForContext(ctx)
}

func NewGenericConfig(data []byte, unmarshaler runtime.Unmarshaler) (Config, error) {
	return internal.NewGenericConfig(data, unmarshaler)
}

func ToGenericConfig(c Config) (*GenericConfig, error) {
	return internal.ToGenericConfig(c)
}

func NewConfigTypeScheme() ConfigTypeScheme {
	return internal.NewConfigTypeScheme(nil)
}

func IsGeneric(cfg Config) bool {
	return internal.IsGeneric(cfg)
}

func ErrNoContext(name string) error {
	return internal.ErrNoContext(name)
}

func IsErrNoContext(err error) bool {
	return cpi.IsErrNoContext(err)
}

func IsErrConfigNotApplicable(err error) bool {
	return cpi.IsErrConfigNotApplicable(err)
}
