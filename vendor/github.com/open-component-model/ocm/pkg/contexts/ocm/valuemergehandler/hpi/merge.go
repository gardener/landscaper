// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package hpi

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/valuemergehandler/handlers/defaultmerge"
	"github.com/open-component-model/ocm/pkg/errors"
)

func AsValue(v interface{}) (*Value, error) {
	if v == nil {
		return nil, nil
	}
	var r Value
	err := r.SetValue(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func GenericMerge[T any](ctx cpi.Context, m *Specification, hint string, local T, inbound T) (bool, T, error) {
	var Nil T

	l, err := AsValue(local)
	if err != nil {
		return false, Nil, err
	}
	t, err := AsValue(inbound)
	if err != nil {
		return false, Nil, err
	}
	mod, err := Merge(ctx, m, "", *l, t)
	if err != nil {
		return false, Nil, err
	}
	if mod {
		inbound = Nil
		err = t.GetValue(&inbound)
		if err != nil {
			return false, Nil, errors.Wrapf(err, "cannot value merge result")
		}
	}
	return mod, inbound, nil
}

// Merge merges two value using the given merge specification.
// The hint describes a merge hint if no algorithm is specified.
// It used the format <string>[@<version>]. If used the is looks
// for an assignment for this hint, first with version and the without version.
func Merge(ctx cpi.Context, m *Specification, hint string, local Value, inbound *Value) (bool, error) {
	var err error

	if m == nil {
		m = &Specification{}
	}
	if m.Algorithm == "" && hint != "" {
		spec := ctx.LabelMergeHandlers().GetAssignment(hint)
		if spec == nil {
			idx := strings.LastIndex(hint, "@")
			if idx > 1 {
				hint = hint[:idx]
			}
			spec = ctx.LabelMergeHandlers().GetAssignment(hint)
		}
		if spec != nil {
			m.Algorithm = spec.Algorithm
			if len(m.Config) == 0 {
				m.Config = spec.Config
			}
		}
	}
	if m.Algorithm == "" {
		m.Algorithm = defaultmerge.ALGORITHM
	}

	h := ctx.LabelMergeHandlers().GetHandler(m.Algorithm)
	if h == nil {
		return false, errors.ErrUnknown(KIND_VALUE_MERGE_ALGORITHM, m.Algorithm)
	}

	Log.Trace("merge handler", "handler", m.Algorithm, "config", m.Config)
	var cfg cpi.ValueMergeHandlerConfig
	if len(m.Config) != 0 {
		cfg, err = h.DecodeConfig(m.Config)
		if err == nil {
			err = cfg.Complete(ctx)
		}
		if err != nil {
			return false, errors.Wrapf(err, "invalid merge config for algorithm %q", m.Algorithm)
		}
	}
	return h.Merge(ctx, local, inbound, cfg)
}
