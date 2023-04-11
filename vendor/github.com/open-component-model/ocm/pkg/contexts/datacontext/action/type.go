// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package action

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/api"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const KIND_ACTION = api.KIND_ACTION

type (
	Selector     = api.Selector
	Action       = api.Action
	ActionSpec   = api.ActionSpec
	ActionResult = api.ActionResult
	ActionType   = api.ActionType
)

////////////////////////////////////////////////////////////////////////////////

func GetAction(kind string) Action {
	return api.GetAction(kind)
}

func EncodeActionSpec(s ActionSpec) ([]byte, error) {
	return api.EncodeActionSpec(s, runtime.DefaultJSONEncoding)
}

func DecodeActionSpec(data []byte) (ActionSpec, error) {
	return api.DecodeActionSpec(data, runtime.DefaultYAMLEncoding)
}

func EncodeActionResult(s ActionResult) ([]byte, error) {
	return api.EncodeActionResult(s, runtime.DefaultJSONEncoding)
}

func DecodeActionResult(data []byte) (ActionResult, error) {
	return api.DecodeActionResult(data, runtime.DefaultYAMLEncoding)
}

func SupportedActionVersions(name string) []string {
	return api.SupportedActionVersions(name)
}
