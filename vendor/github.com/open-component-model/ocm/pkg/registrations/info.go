// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registrations

import (
	"strings"

	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/listformat"
)

type HandlerInfos []HandlerInfo

var _ listformat.ListElements = HandlerInfos(nil)

func (h HandlerInfos) Size() int {
	return len(h)
}

func (h HandlerInfos) Key(i int) string {
	return h[i].Name
}

func (h HandlerInfos) Description(i int) string {
	var desc string

	if h[i].Node {
		desc = "[" + generics.Conditional(h[i].ShortDesc == "", "intermediate", strings.Trim(h[i].ShortDesc, "\n")) + "]"
	} else {
		desc = h[i].ShortDesc
	}
	return desc + generics.Conditional(h[i].Description == "", "", "\n\n"+strings.Trim(h[i].Description, "\n"))
}

type HandlerInfo struct {
	Name        string
	ShortDesc   string
	Description string
	Node        bool
}

func NewLeafHandlerInfo(short, desc string) HandlerInfos {
	return HandlerInfos{
		{
			ShortDesc:   short,
			Description: desc,
		},
	}
}

func NewNodeHandlerInfo(short, desc string) HandlerInfos {
	return HandlerInfos{
		{
			ShortDesc:   short,
			Description: desc,
			Node:        true,
		},
	}
}
