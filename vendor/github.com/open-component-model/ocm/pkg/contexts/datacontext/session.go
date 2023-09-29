// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

// Session is a context keeping track of objects requiring a close
// after final use. When closing a session all subsequent objects
// will be closed in the opposite order they are added.
// Added closers may be closed prio to the session without causing
// errors.
type Session interface {
	Closer(closer io.Closer, extra ...interface{}) (io.Closer, error)
	GetOrCreate(key interface{}, creator func(SessionBase) Session) Session
	AddCloser(closer io.Closer, callbacks ...accessio.CloserCallback) io.Closer
	Close() error
	IsClosed() bool
}

type SessionBase interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()

	Session() Session
	IsClosed() bool
	AddCloser(closer io.Closer, callbacks ...accessio.CloserCallback) io.Closer
}

type ObjectKey struct {
	Object interface{}
	Name   string
}

type session struct {
	base sessionBase
}

type sessionBase struct {
	sync.RWMutex
	session  Session
	closed   bool
	closer   []io.Closer
	sessions map[interface{}]Session
}

func NewSession() Session {
	s := &session{
		sessionBase{
			sessions: map[interface{}]Session{},
		},
	}
	s.base.session = s
	return s
}

func GetOrCreateSubSession(s Session, key interface{}, creator func(SessionBase) Session) Session {
	if s == nil {
		s = NewSession()
	}
	return s.GetOrCreate(key, creator)
}

func (s *session) IsClosed() bool {
	s.base.RLock()
	defer s.base.RUnlock()
	return s.base.closed
}

func (s *session) Close() error {
	s.base.Lock()
	defer s.base.Unlock()
	return s.base.Close()
}

func (s *session) Closer(closer io.Closer, extra ...interface{}) (io.Closer, error) {
	for _, e := range extra {
		if err, ok := e.(error); ok && err != nil {
			return nil, err
		}
	}
	if closer == nil {
		return nil, nil
	}
	s.base.Lock()
	defer s.base.Unlock()
	s.base.AddCloser(closer)

	return closer, nil
}

func (s *session) AddCloser(closer io.Closer, callbacks ...accessio.CloserCallback) io.Closer {
	if closer == nil {
		return nil
	}
	s.base.Lock()
	defer s.base.Unlock()
	return s.base.AddCloser(closer, callbacks...)
}

func (s *session) GetOrCreate(key interface{}, creator func(SessionBase) Session) Session {
	s.base.Lock()
	defer s.base.Unlock()
	return s.base.GetOrCreate(key, creator)
}

func (s *sessionBase) Session() Session {
	return s.session
}

func (s *sessionBase) IsClosed() bool {
	return s.closed
}

func (s *sessionBase) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	list := errors.ErrListf("closing session")
	for i := len(s.closer) - 1; i >= 0; i-- {
		list.Add(s.closer[i].Close())
	}
	return list.Result()
}

func (s *sessionBase) AddCloser(closer io.Closer, callbacks ...accessio.CloserCallback) io.Closer {
	s.closer = append(s.closer, accessio.OnceCloser(closer, callbacks...))
	return closer
}

func (s *sessionBase) GetOrCreate(key interface{}, creator func(SessionBase) Session) Session {
	cur := s.sessions[key]
	if cur == nil {
		cur = creator(s)
		s.sessions[key] = cur
	}
	return cur
}
