// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strconv"
	"time"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/generics"
)

// ParseDeltaTime parses a time diff relative to the actual
// time and returns the resulting time.
func ParseDeltaTime(s string, past bool) (time.Time, error) {
	var t time.Time

	f := int64(generics.Conditional(past, -1, 1))

	if len(s) < 2 {
		return t, errors.Newf("invalid time diff %q", s)
	}
	i, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return t, errors.Wrapf(err, "invalid time diff %q", s)
	}

	d := scale[s[len(s)-1:]]
	if d == nil {
		return t, errors.Newf("invalid time diff %q", s)
	}
	return d(i*f, time.Now()), nil
}

type timeModifier func(d int64, t time.Time) time.Time

var scale = map[string]timeModifier{
	"s": func(d int64, t time.Time) time.Time { return t.Add(time.Duration(d) * time.Second) },
	"m": func(d int64, t time.Time) time.Time { return t.Add(time.Duration(d) * time.Minute) },
	"h": func(d int64, t time.Time) time.Time { return t.Add(time.Duration(d) * time.Hour) },
	"d": func(d int64, t time.Time) time.Time { return t.AddDate(0, 0, int(d)) },
	"M": func(d int64, t time.Time) time.Time { return t.AddDate(0, int(d), 0) },
	"y": func(d int64, t time.Time) time.Time { return t.AddDate(int(d), 0, 0) },
}
