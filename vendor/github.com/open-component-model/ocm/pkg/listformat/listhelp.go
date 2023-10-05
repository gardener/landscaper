// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package listformat

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/utils"
)

type StringElementDescriptionList []string

func (l StringElementDescriptionList) Size() int                { return len(l) / 2 }
func (l StringElementDescriptionList) Key(i int) string         { return l[2*i] }
func (l StringElementDescriptionList) Description(i int) string { return l[2*i+1] }

type StringElementList []string

func (l StringElementList) Size() int                { return len(l) }
func (l StringElementList) Key(i int) string         { return l[i] }
func (l StringElementList) Description(i int) string { return "" }

func FormatList(def string, elems ...string) string {
	return FormatListElements(def, StringElementList(elems))
}

type maplist[E any] struct {
	desc func(E) string
	keys []string
	m    map[string]E
}

func (l *maplist[E]) Size() int                { return len(l.keys) }
func (l *maplist[E]) Key(i int) string         { return l.keys[i] }
func (l *maplist[E]) Description(i int) string { return l.desc(l.m[l.keys[i]]) }

func FormatMapElements[E any](def string, m map[string]E, desc ...func(E) string) string {
	if len(desc) == 0 || desc[0] == nil {
		desc = []func(E) string{StringDescription[E]}
	}
	keys := utils.StringMapKeys(m)
	return FormatListElements(def, &maplist[E]{
		desc: desc[0],
		keys: keys,
		m:    m,
	})
}

type DescriptionSource interface {
	GetDescription() string
}

type DirectDescriptionSource interface {
	Description() string
}

func StringDescription[E any](e E) string {
	if d, ok := any(e).(DescriptionSource); ok {
		return d.GetDescription()
	}
	if d, ok := any(e).(DirectDescriptionSource); ok {
		return d.Description()
	}
	return fmt.Sprintf("%s", any(e))
}

type ListElements interface {
	Size() int
	Key(i int) string
	Description(i int) string
}

func FormatListElements(def string, elems ListElements) string {
	names := ""
	size := elems.Size()

	for i := 0; i < size; i++ {
		key := elems.Key(i)
		names = fmt.Sprintf("%s  - <code>%s</code>", names, key)
		if key == def {
			names += " (default)"
		}
		desc := elems.Description(i)
		if desc != "" {
			names += ": " + utils.IndentLines(desc, "    ", true)
			if strings.Contains(desc, "\n") {
				names += "\n"
			}
		}
		names += "\n"
	}
	return names
}
