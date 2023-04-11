// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"reflect"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/runtime/scheme"
)

const KIND_ACTION = "action"

type ActionTypeRegistry interface {
	RegisterAction(name string, specproto ActionSpec, resultproto ActionResult, description string, attrs ...string) error
	RegisterActionType(name string, typ ActionType) error

	DecodeActionSpec(data []byte, unmarshaler runtime.Unmarshaler) (ActionSpec, error)
	EncodeActionSpec(spec ActionSpec, marshaler runtime.Marshaler) ([]byte, error)

	DecodeActionResult(data []byte, unmarshaler runtime.Unmarshaler) (ActionResult, error)
	EncodeActionResult(spec ActionResult, marshaler runtime.Marshaler) ([]byte, error)

	GetAction(name string) Action
	SupportedActionVersions(name string) []string
}

type action struct {
	name        string
	description string
	attributes  []string
	specproto   reflect.Type
	resultproto reflect.Type
}

var _ Action = (*action)(nil)

func (a *action) Name() string {
	return a.name
}

func (a *action) Description() string {
	return a.description
}

func (a *action) ConsumerAttributes() []string {
	return a.attributes
}

func (a *action) SpecificationProto() reflect.Type {
	return a.specproto
}

func (a *action) ResultProto() reflect.Type {
	return a.resultproto
}

type actionRegistry struct {
	lock        sync.Mutex
	actions     map[string]*action
	actionspecs scheme.Scheme[ActionSpec, ActionSpecType]
	resultspecs scheme.Scheme[ActionResult, ActionResultType]
}

func NewActionTypeRegistry() ActionTypeRegistry {
	return &actionRegistry{
		actions:     map[string]*action{},
		actionspecs: scheme.NewScheme[ActionSpec, ActionSpecType](),
		resultspecs: scheme.NewScheme[ActionResult, ActionResultType](),
	}
}

func (r *actionRegistry) RegisterAction(name string, specproto ActionSpec, resultproto ActionResult, description string, attrs ...string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	ai := r.actions[name]
	if ai != nil {
		return errors.ErrAlreadyExists(KIND_ACTION, name)
	}
	st := reflect.TypeOf(specproto)
	for st.Kind() == reflect.Ptr {
		st = st.Elem()
	}
	rt := reflect.TypeOf(specproto)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	ai = &action{
		name:        name,
		description: description,
		attributes:  append(attrs[:0:0], attrs...),
		specproto:   st,
		resultproto: rt,
	}
	r.actions[name] = ai
	return nil
}

func (r *actionRegistry) RegisterActionType(name string, typ ActionType) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	k, _ := runtime.KindVersion(name)

	ai := r.actions[k]
	if ai == nil {
		return errors.ErrNotFound(KIND_ACTION, k)
	}

	err := r.actionspecs.RegisterType(name, typ.SpecificationType())
	if err != nil {
		return err
	}

	err = r.resultspecs.RegisterType(name, typ.ResultType())
	return err
}

func (r *actionRegistry) GetAction(name string) Action {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.actions[name]
}

func (r *actionRegistry) DecodeActionSpec(data []byte, unmarshaler runtime.Unmarshaler) (ActionSpec, error) {
	return r.actionspecs.Decode(data, unmarshaler)
}

func (r *actionRegistry) DecodeActionResult(data []byte, unmarshaler runtime.Unmarshaler) (ActionResult, error) {
	return r.resultspecs.Decode(data, unmarshaler)
}

func (r *actionRegistry) EncodeActionSpec(spec ActionSpec, marshaler runtime.Marshaler) ([]byte, error) {
	return r.actionspecs.Encode(spec, marshaler)
}

func (r *actionRegistry) EncodeActionResult(spec ActionResult, marshaler runtime.Marshaler) ([]byte, error) {
	return r.resultspecs.Encode(spec, marshaler)
}

func (r *actionRegistry) SupportedActionVersions(name string) []string {
	return r.actionspecs.KnownVersions(name)
}

////////////////////////////////////////////////////////////////////////////////

var registry = NewActionTypeRegistry()

func RegisterAction(name string, specproto ActionSpec, resultproto ActionResult, description string, attrs ...string) error {
	return registry.RegisterAction(name, specproto, resultproto, description, attrs...)
}

func RegisterType(kind string, version string, typ ActionType) error {
	return registry.RegisterActionType(runtime.TypeName(kind, version), typ)
}

func GetAction(name string) Action {
	return registry.GetAction(name)
}

func DecodeActionSpec(data []byte, unmarshaler runtime.Unmarshaler) (ActionSpec, error) {
	return registry.DecodeActionSpec(data, unmarshaler)
}

func EncodeActionSpec(spec ActionSpec, marshaler runtime.Marshaler) ([]byte, error) {
	return registry.EncodeActionSpec(spec, marshaler)
}

func DecodeActionResult(data []byte, unmarshaler runtime.Unmarshaler) (ActionResult, error) {
	return registry.DecodeActionResult(data, unmarshaler)
}

func EncodeActionResult(spec ActionResult, marshaler runtime.Marshaler) ([]byte, error) {
	return registry.EncodeActionResult(spec, marshaler)
}

func SupportedActionVersions(name string) []string {
	return registry.SupportedActionVersions(name)
}
