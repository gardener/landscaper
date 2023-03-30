// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"encoding/json"
	"sync"

	"github.com/mandelsoft/logging"
	logcfg "github.com/mandelsoft/logging/config"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/errors"
)

// REALM is used to tag all logging done by this library with the ocm tag.
// This is also used as message context to configure settings for all
// log output provided by this library.
var REALM = logging.NewRealm("ocm")

type StaticContext struct {
	logging.Context
	applied map[string]struct{}
	lock    sync.Mutex
}

func NewContext(ctx logging.Context) *StaticContext {
	if ctx == nil {
		ctx = logging.DefaultContext()
	}
	return &StaticContext{
		Context: ctx.WithContext(REALM),
		applied: map[string]struct{}{},
	}
}

// Configure applies a configuration once.
// Every config identified by its hash is applied
// only once.
func (s *StaticContext) Configure(config *logcfg.Config, extra ...string) error {
	add := ""
	for _, e := range extra {
		if e != "" {
			add += "/" + e
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal log config")
	}
	d := digest.FromBytes(data).String() + add

	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.applied[d]; ok {
		return nil
	}
	s.applied[d] = struct{}{}
	return logcfg.Configure(logContext, config)
}

var logContext = NewContext(nil)

// SetContext sets a new precondigure context.
// This function should be called prior to any configuration
// to avoid loosing them.
func SetContext(ctx logging.Context) {
	logContext = NewContext(ctx)
}

// Context returns the default logging configuration used for this library.
func Context() *StaticContext {
	return logContext
}

// Logger determines a default logger for this given message context
// based on the rule settings for this library.
func Logger(messageContext ...logging.MessageContext) logging.Logger {
	return logContext.Logger(messageContext...)
}

// Configure applies configuration for the default log context
// provided by this package.
func Configure(config *logcfg.Config, extra ...string) error {
	return logContext.Configure(config, extra...)
}
