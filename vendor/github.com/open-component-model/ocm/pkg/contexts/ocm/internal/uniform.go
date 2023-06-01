// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"
)

// UniformRepositorySpec is generic specification of the repository
// for handling as part of standard references.
type UniformRepositorySpec struct {
	// Type
	Type string `json:"type,omitempty"`
	// Host is the hostname of an ocm ref.
	Host string `json:"host,omitempty"`
	// SubPath is the sub path spec used to host component versions
	SubPath string `json:"subPath,omitempty"`
	// Info is the file path used to host ctf component versions
	Info string `json:"filePath,omitempty"`

	// CreateIfMissing indicates whether a file based or dynamic repo should be created if it does not exist
	CreateIfMissing bool `json:"createIfMissing,omitempty"`
	// TypeHintshould be set if CreateIfMissing is true to help to decide what kind of repo to create
	TypeHint string `json:"typeHint,omitempty"`
}

// CredHost fallback to legacy docker domain if applicable
// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674.
func (r *UniformRepositorySpec) CredHost() string {
	if r.Host == dockerHubDomain {
		return dockerHubLegacyDomain
	}
	return r.Host
}

func (u *UniformRepositorySpec) String() string {
	t := u.Type
	if t != "" {
		t += "::"
	}
	if u.Info != "" {
		return fmt.Sprintf("%s%s", t, u.Info)
	} else {
		s := u.SubPath
		if s != "" {
			s = "/" + s
		}
		return fmt.Sprintf("%s%s%s", t, u.Host, s)
	}
}

func UniformRepositorySpecForUnstructured(un *runtime.UnstructuredVersionedTypedObject) *UniformRepositorySpec {
	m := un.Object.FlatCopy()
	delete(m, runtime.ATTR_TYPE)

	d, err := json.Marshal(m)
	if err != nil {
		logrus.Error(err)
	}

	return &UniformRepositorySpec{Type: un.Type, Info: string(d)}
}

type RepositorySpecHandler interface {
	MapReference(ctx Context, u *UniformRepositorySpec) (RepositorySpec, error)
}

type RepositorySpecHandlers interface {
	Register(hdlr RepositorySpecHandler, types ...string)
	Copy() RepositorySpecHandlers
	KnownTypeNames() []string
	GetHandlers(typ string) []RepositorySpecHandler
	MapUniformRepositorySpec(ctx Context, u *UniformRepositorySpec) (RepositorySpec, error)
}

var DefaultRepositorySpecHandlers = NewRepositorySpecHandlers()

func RegisterRepositorySpecHandler(hdlr RepositorySpecHandler, types ...string) {
	DefaultRepositorySpecHandlers.Register(hdlr, types...)
}

type specHandlers struct {
	lock     sync.RWMutex
	handlers map[string][]RepositorySpecHandler
}

func NewRepositorySpecHandlers() RepositorySpecHandlers {
	return &specHandlers{handlers: map[string][]RepositorySpecHandler{}}
}

func (s *specHandlers) Register(hdlr RepositorySpecHandler, types ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if hdlr != nil {
		for _, typ := range types {
			s.handlers[typ] = append(s.handlers[typ], hdlr)
		}
	}
}

func (s *specHandlers) Copy() RepositorySpecHandlers {
	s.lock.RLock()
	defer s.lock.RUnlock()

	n := NewRepositorySpecHandlers().(*specHandlers)
	for typ, hdlrs := range s.handlers {
		n.handlers[typ] = append(hdlrs[:0:0], hdlrs...)
	}
	return n
}

func (s *specHandlers) KnownTypeNames() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return utils.StringMapKeys(s.handlers)
}

func (s *specHandlers) GetHandlers(typ string) []RepositorySpecHandler {
	s.lock.RLock()
	defer s.lock.RUnlock()

	hdlrs := s.handlers[typ]
	if len(hdlrs) == 0 {
		return nil
	}
	result := make([]RepositorySpecHandler, len(hdlrs))
	copy(result, hdlrs)
	return result
}

func (s *specHandlers) MapUniformRepositorySpec(ctx Context, u *UniformRepositorySpec) (RepositorySpec, error) {
	var err error
	s.lock.RLock()
	defer s.lock.RUnlock()

	deferr := errors.ErrNotSupported("uniform repository ref", u.String())
	if u.Type == "" {
		if u.Info != "" {
			spec := ctx.GetAlias(u.Info)
			if spec != nil {
				return spec, nil
			}
			deferr = errors.ErrUnknown("repository", u.Info)
		}
		if u.Host != "" {
			spec := ctx.GetAlias(u.Host)
			if spec != nil {
				return spec, nil
			}
			deferr = errors.ErrUnknown("repository", u.Host)
		}
	}

	spec, err := s.handle(ctx, u.Type, u)
	if err != nil || spec != nil {
		return spec, err
	}

	if u.Info != "" {
		spec := &runtime.UnstructuredVersionedTypedObject{}
		err = runtime.DefaultJSONEncoding.Unmarshal([]byte(u.Info), spec)
		if err == nil {
			if spec.GetType() == spec.GetKind() && spec.GetVersion() == "v1" { // only type set, use it as version
				spec.SetType(u.Type + runtime.VersionSeparator + spec.GetType())
			}
			if spec.GetKind() != u.Type {
				return nil, errors.ErrInvalid()
			}
			return ctx.RepositoryTypes().Convert(spec)
		}
	}

	spec, err = s.handle(ctx, "*", u)
	if spec == nil && err == nil {
		err = deferr
	}
	return spec, err
}

func (s *specHandlers) handle(ctx Context, typ string, u *UniformRepositorySpec) (RepositorySpec, error) {
	for _, h := range s.handlers[typ] {
		spec, err := h.MapReference(ctx, u)
		if err != nil || spec != nil {
			return spec, err
		}
	}
	return nil, nil
}
