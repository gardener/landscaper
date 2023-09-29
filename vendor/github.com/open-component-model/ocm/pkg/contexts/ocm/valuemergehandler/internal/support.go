// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type ConfigPointer[T any] interface {
	Config
	*T
}

type Merger[C, T any] func(ctx Context, cfg C, local T, target *T) (bool, error)

func New[C any, L any, P ConfigPointer[C]](algo string, desc string, merger Merger[P, L]) Handler {
	return &HandlerSupport[C, L, P]{
		algorithm:   algo,
		description: desc,
		merger:      merger,
	}
}

// HandlerSupport is a basic support for label merge handlers.
type HandlerSupport[C any, L any, P ConfigPointer[C]] struct {
	algorithm   string
	description string
	merger      Merger[P, L]
}

func (h *HandlerSupport[C, L, P]) Algorithm() string {
	return h.algorithm
}

func (h *HandlerSupport[C, L, P]) Description() string {
	return h.description
}

func (h HandlerSupport[C, L, P]) DecodeConfig(data []byte) (Config, error) {
	var cfg C
	err := runtime.DefaultYAMLEncoding.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	var p P = &cfg
	return p, nil
}

func (h *HandlerSupport[C, L, P]) Merge(ctx Context, local Value, inbound *Value, cfg Config) (bool, error) {
	var c P

	if cfg == nil {
		var zero C
		c = &zero
		err := c.Complete(ctx)
		if err != nil {
			return false, errors.Wrapf(err, "[%s] invalid initial config")
		}
	} else {
		var ok bool

		c, ok = cfg.(P)
		if !ok {
			return false, errors.ErrInvalid("[%s] value merge config type", h.algorithm, fmt.Sprintf("%T", cfg))
		}
	}

	var lv L
	if err := local.GetValue(&lv); err != nil {
		return false, errors.Wrapf(err, "[%s] local value is not valid", h.algorithm)
	}

	var tv L
	if err := inbound.GetValue(&tv); err != nil {
		return false, errors.Wrapf(err, "[%s] inbound value is not valid", h.algorithm)
	}

	modified, err := h.merger(ctx, c, lv, &tv)
	if err != nil {
		return false, errors.Wrapf(err, "[%s]", h.algorithm)
	}

	if modified {
		inbound.SetValue(tv)
	}
	return modified, nil
}
