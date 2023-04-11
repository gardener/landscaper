// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
)

type _consumers struct {
	sync.RWMutex
	data map[string]*_consumer
}

func newConsumers() *_consumers {
	return &_consumers{
		data: map[string]*_consumer{},
	}
}

func (c *_consumers) Get(id ConsumerIdentity) *_consumer {
	c.RLock()
	defer c.RUnlock()
	return c.data[string(id.Key())]
}

func (c *_consumers) Set(id ConsumerIdentity, creds CredentialsSource) {
	c.Lock()
	defer c.Unlock()
	c.data[string(id.Key())] = &_consumer{
		identity:    id,
		credentials: creds,
	}
}

// Match matches a given request (pattern) against configured
// identities.
func (c *_consumers) Match(pattern ConsumerIdentity, m IdentityMatcher) *_consumer {
	c.RLock()
	defer c.RUnlock()
	var found *_consumer
	var cur ConsumerIdentity
	for _, s := range c.data {
		if m(pattern, cur, s.identity) {
			found = s
			cur = s.identity
		}
	}
	return found
}

type _consumer struct {
	identity    ConsumerIdentity
	credentials CredentialsSource
}

func (c *_consumer) GetCredentials() CredentialsSource {
	return c.credentials
}
