// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"context"
	"fmt"
)

// ForContextByKey retrieves the context for a given key to be used for a context.Context.
// If not defined, it returns the given default context and false.
func ForContextByKey(ctx context.Context, key interface{}, def Context) (Context, bool) {
	c := ctx.Value(key)
	if c == nil {
		return def, false
	}
	return c.(Context), true
}

type ElementCopyable[T any] interface {
	comparable
	Copy() T
}

type ElementCreator[T any] func(base ...T) T

func SetupElement[T ElementCopyable[T]](mode BuilderMode, target *T, create ElementCreator[T], def T) error {
	var zero T
	if *target == zero {
		switch mode {
		case MODE_INITIAL:
			*target = create()
		case MODE_CONFIGURED:
			*target = def.Copy()
		case MODE_EXTENDED:
			*target = create(def)
		case MODE_DEFAULTED:
			fallthrough
		case MODE_SHARED:
			*target = def
		default:
			return fmt.Errorf("invalid context creation mode %s", mode)
		}
	}

	return nil
}
