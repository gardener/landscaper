// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package regex

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Alpha defines the alpha atom.
	// This only allows upper and lower case characters.
	Alpha = Match(`[A-Za-z]+`)

	// Numeric defines the alpha atom.
	// This only allows a non-empty sequence of digits.
	Numeric = Match(`[0-9]+`)

	// AlphaNumeric defines the alpha numeric atom, typically a
	// component of names. This only allows upper and lower case characters and digits.
	AlphaNumeric = Match(`[A-Za-z0-9]+`)

	// Identifier is an AlphaNumeric regexp starting with an Alpha regexp.
	Identifier = Sequence(Alpha, Match(`[A-Za-z0-9]`), Optional(Literal("+"), Alpha))
)

// Match compiles the string to a regular expression.
var Match = regexp.MustCompile

// Literal compiles s into a literal regular expression, escaping any regexp
// reserved characters.
func Literal(s string) *regexp.Regexp {
	re := Match(regexp.QuoteMeta(s))

	if _, complete := re.LiteralPrefix(); !complete {
		panic("must be a literal")
	}

	return re
}

const classBytes = `]\`

func quoteCharClass(s string) string {
	res := ""
	for _, r := range s {
		if strings.Contains(classBytes, string(r)) {
			res += "\\"
		}
		res += string(r)
	}
	return res
}

// CharSet compiles a set of matching charaters.
func CharSet(s string) *regexp.Regexp {
	return Match("[" + quoteCharClass(s) + "]")
}

// Sequence defines a full expression, where each regular expression must
// follow the previous.
func Sequence(res ...*regexp.Regexp) *regexp.Regexp {
	var s string
	for _, re := range res {
		s += re.String()
	}

	return Match(s)
}

// Optional wraps the expression in a non-capturing group and makes the
// production optional.
func Optional(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(Group(res...).String() + `?`)
}

// Repetition wraps the regexp in a non-capturing group to get a range of
// matches.
func Repetition(min, max int, res ...*regexp.Regexp) *regexp.Regexp {
	return Match(Group(res...).String() + fmt.Sprintf(`{%d,%d}`, min, max))
}

// Repeated wraps the regexp in a non-capturing group to get one or more
// matches.
func Repeated(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(Group(res...).String() + `+`)
}

// OptionalRepeated wraps the regexp in a non-capturing group to get any
// number of matches.
func OptionalRepeated(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(Group(res...).String() + `*`)
}

// Group wraps the regexp in a non-capturing group.
func Group(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(`(?:` + Sequence(res...).String() + `)`)
}

// Or wraps alternative regexps.
func Or(res ...*regexp.Regexp) *regexp.Regexp {
	var s string
	sep := ""
	for _, re := range res {
		s += sep + Group(re).String()
		sep = "|"
	}
	return Match(`(?:` + s + `)`)
}

// Capture wraps the expression in a capturing group.
func Capture(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(`(` + Sequence(res...).String() + `)`)
}

// Anchored anchors the regular expression by adding start and end delimiters.
func Anchored(res ...*regexp.Regexp) *regexp.Regexp {
	return Match(`^` + Sequence(res...).String() + `$`)
}
