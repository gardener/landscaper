// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/runtime"
)

type Action interface {
	Name() string
	Description() string
	Usage() string
	ConsumerAttributes() []string
	GetVersion(string) ActionType
	SupportedVersions() []string
}

////////////////////////////////////////////////////////////////////////////////
// Action Specification

type Selector string

func (s Selector) ApplyActionHandlerOptionTo(opts *Options) {
	opts.Selectors = append(opts.Selectors, s)
}

type ActionSpec interface {
	runtime.VersionedTypedObject
	SetVersion(string)
	Selector() Selector
	GetConsumerAttributes() common.Properties
}

type ActionSpecType runtime.VersionedTypedObjectType[ActionSpec]

////////////////////////////////////////////////////////////////////////////////
// Action Result

type ActionResult interface {
	runtime.VersionedTypedObject
	SetVersion(string)
	SetType(string)
	GetMessage() string
}

// CommonResult is the minimal action result.
type CommonResult struct {
	runtime.ObjectVersionedType `json:",inline"`
	Message                     string `json:"message,omitempty"`
}

func (r *CommonResult) GetMessage() string {
	return r.Message
}

func (r *CommonResult) SetType(typ string) {
	r.Type = typ
}

////////////////////////////////////////////////////////////////////////////////
// Action Type

type ActionResultType runtime.VersionedTypedObjectType[ActionResult]

type ActionType interface {
	runtime.VersionedTypedObject
	SpecificationType() ActionSpecType
	ResultType() ActionResultType
}

////////////////////////////////////////////////////////////////////////////////
// Options Type

type Option interface {
	ApplyActionHandlerOptionTo(*Options)
}

type Options struct {
	Action    string
	Selectors []Selector
	Priority  int
	Versions  []string
}

var _ Option = (*Options)(nil)
