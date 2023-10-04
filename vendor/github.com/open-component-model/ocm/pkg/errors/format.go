// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors

type ErrorFormatter interface {
	Format(kind string, elem *string, ctxkind string, ctx string) string
}

type defaultFormatter struct {
	verb        string
	msg         string
	preposition string
}

func NewDefaultFormatter(verb, msg, preposition string) ErrorFormatter {
	if verb != "" {
		verb += " "
	}
	return &defaultFormatter{
		verb:        verb,
		msg:         msg,
		preposition: preposition,
	}
}

func (f *defaultFormatter) Format(kind string, elem *string, ctxkind string, ctx string) string {
	if ctx != "" {
		if ctxkind != "" {
			ctx = ctxkind + " " + ctx
		}
		ctx = " " + f.preposition + " " + ctx
	}
	elems := ""
	if elem != nil {
		elems = "\"" + *elem + "\" "
	}
	if kind != "" {
		kind += " "
	}
	if kind == "" && elems == "" {
		return f.msg + ctx
	}
	return kind + elems + f.verb + f.msg + ctx
}
