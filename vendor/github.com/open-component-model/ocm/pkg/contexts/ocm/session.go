// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
)

type ComponentContainer interface {
	LookupComponent(name string) (ComponentAccess, error)
}
type ComponentVersionContainer interface {
	LookupVersion(version string) (ComponentVersionAccess, error)
}

type EvaluationResult struct {
	Ref        RefSpec
	Repository Repository
	Component  ComponentAccess
	Version    ComponentVersionAccess
}

type Session interface {
	datacontext.Session

	Finalize(Finalizer)
	LookupRepository(Context, RepositorySpec) (Repository, error)
	LookupComponent(ComponentContainer, string) (ComponentAccess, error)
	LookupComponentVersion(r Repository, comp, vers string) (ComponentVersionAccess, error)
	GetComponentVersion(ComponentVersionContainer, string) (ComponentVersionAccess, error)
	EvaluateRef(ctx Context, ref string) (*EvaluationResult, error)
	EvaluateComponentRef(ctx Context, ref string) (*EvaluationResult, error)
	EvaluateVersionRef(ctx Context, ref string) (*EvaluationResult, error)
	DetermineRepository(ctx Context, ref string) (Repository, UniformRepositorySpec, error)
	DetermineRepositoryBySpec(ctx Context, spec *UniformRepositorySpec) (Repository, error)
}

type session struct {
	datacontext.Session
	base         datacontext.SessionBase
	repositories map[datacontext.ObjectKey]Repository
	components   map[datacontext.ObjectKey]ComponentAccess
	versions     map[datacontext.ObjectKey]ComponentVersionAccess
}

var _ Session = (*session)(nil)

var key = reflect.TypeOf(session{})

func NewSession(s datacontext.Session) Session {
	return datacontext.GetOrCreateSubSession(s, key, newSession).(Session)
}

func newSession(s datacontext.SessionBase) datacontext.Session {
	return &session{
		Session:      s.Session(),
		base:         s,
		repositories: map[datacontext.ObjectKey]Repository{},
		components:   map[datacontext.ObjectKey]ComponentAccess{},
		versions:     map[datacontext.ObjectKey]ComponentVersionAccess{},
	}
}

type Finalizer interface {
	Finalize() error
}

type finalizer struct {
	finalizer Finalizer
}

func (f *finalizer) Close() error {
	return f.finalizer.Finalize()
}

func (s *session) Finalize(f Finalizer) {
	s.Session.AddCloser(&finalizer{f})
}

func (s *session) Close() error {
	return s.Session.Close()
	// TODO: cleanup cache
}

func (s *session) LookupRepository(ctx Context, spec RepositorySpec) (Repository, error) {
	spec, err := ctx.RepositoryTypes().Convert(spec)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	key := datacontext.ObjectKey{
		Object: ctx,
		Name:   string(data),
	}

	s.base.Lock()
	defer s.base.Unlock()
	if s.base.IsClosed() {
		return nil, errors.ErrClosed("session")
	}

	if r := s.repositories[key]; r != nil {
		return r, nil
	}
	repo, err := ctx.RepositoryForSpec(spec)
	if err != nil {
		return nil, err
	}
	s.repositories[key] = repo
	s.base.AddCloser(repo)
	return repo, err
}

func (s *session) LookupComponent(c ComponentContainer, name string) (ComponentAccess, error) {
	key := datacontext.ObjectKey{
		Object: c,
		Name:   name,
	}
	s.base.Lock()
	defer s.base.Unlock()
	if s.base.IsClosed() {
		return nil, errors.ErrClosed("session")
	}
	if ns := s.components[key]; ns != nil {
		return ns, nil
	}
	ns, err := c.LookupComponent(name)
	if err != nil {
		return nil, err
	}
	s.components[key] = ns
	s.base.AddCloser(ns)
	return ns, err
}

func (s *session) LookupComponentVersion(r Repository, comp, vers string) (ComponentVersionAccess, error) {
	component, err := s.LookupComponent(r, comp)
	if err != nil {
		return nil, err
	}
	return s.GetComponentVersion(component, vers)
}

func (s *session) GetComponentVersion(c ComponentVersionContainer, version string) (ComponentVersionAccess, error) {
	if c == nil {
		return nil, fmt.Errorf("no container given")
	}
	key := datacontext.ObjectKey{
		Object: c,
		Name:   version,
	}
	s.base.Lock()
	defer s.base.Unlock()
	if s.base.IsClosed() {
		return nil, errors.ErrClosed("session")
	}
	if obj := s.versions[key]; obj != nil {
		return obj, nil
	}
	obj, err := c.LookupVersion(version)
	if err != nil {
		return nil, err
	}
	s.versions[key] = obj
	s.base.AddCloser(obj)
	return obj, err
}

func (s *session) EvaluateVersionRef(ctx Context, ref string) (*EvaluationResult, error) {
	evaluated, err := s.EvaluateComponentRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	versions, err := evaluated.Component.ListVersions()
	if err != nil {
		return evaluated, errors.Wrapf(err, "%s[%s]: listing versions", ref, evaluated.Ref.Component)
	}
	if len(versions) != 1 {
		return evaluated, errors.Wrapf(err, "%s {%s]: found %d components", ref, evaluated.Ref.Component, len(versions))
	}
	evaluated.Version, err = s.GetComponentVersion(evaluated.Component, versions[0])
	if err != nil {
		return evaluated, errors.Wrapf(err, "%s {%s:%s]: listing components", ref, evaluated.Ref.Component, versions[0])
	}
	evaluated.Ref.Version = &versions[0]
	return evaluated, nil
}

func (s *session) EvaluateComponentRef(ctx Context, ref string) (*EvaluationResult, error) {
	evaluated, err := s.EvaluateRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	if evaluated.Component == nil {
		lister := evaluated.Repository.ComponentLister()
		if lister == nil {
			return evaluated, errors.Newf("%s: no component specified", ref)
		}
		if n, err := lister.NumComponents(""); n != 1 {
			if err != nil {
				return evaluated, errors.Wrapf(err, "%s: listing components", ref)
			}
			// return evaluated, errors.Newf("%s: found %d components", ref, n)
			return evaluated, nil // return repo ref
		}
		list, err := lister.GetComponents("", true)
		if err != nil {
			return evaluated, errors.Wrapf(err, "%s: listing components", ref)
		}
		evaluated.Ref.Component = list[0]
		evaluated.Component, err = s.LookupComponent(evaluated.Repository, list[0])
		if err != nil {
			return evaluated, errors.Wrapf(err, "%s: listing components", ref)
		}
	}
	return evaluated, nil
}

func (s *session) EvaluateRef(ctx Context, ref string) (*EvaluationResult, error) {
	var err error
	result := &EvaluationResult{}
	result.Ref, err = ParseRef(ref)
	if err != nil {
		return nil, err
	}

	result.Repository, err = s.DetermineRepositoryBySpec(ctx, &result.Ref.UniformRepositorySpec)
	if err != nil {
		return result, err
	}
	if result.Ref.Component != "" {
		result.Component, err = s.LookupComponent(result.Repository, result.Ref.Component)
		if err != nil {
			return nil, err
		}
		if result.Ref.IsVersion() {
			result.Version, err = s.GetComponentVersion(result.Component, *result.Ref.Version)
		}
	}
	return result, err
}

func (s *session) DetermineRepository(ctx Context, ref string) (Repository, UniformRepositorySpec, error) {
	spec, err := ParseRepo(ref)
	if err != nil {
		return nil, spec, err
	}
	r, err := s.DetermineRepositoryBySpec(ctx, &spec)
	return r, spec, err
}

func (s *session) DetermineRepositoryBySpec(ctx Context, spec *UniformRepositorySpec) (Repository, error) {
	rspec, err := ctx.MapUniformRepositorySpec(spec)
	if err != nil {
		return nil, err
	}
	return s.LookupRepository(ctx, rspec)
}
