// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sort"
	"sync"

	"github.com/open-component-model/ocm/pkg/generics"
)

// UsageContext descibes a dediacetd type specific
// sub usage kinds for an object requiring credentials.
// For example, for an object providing a hierarchical
// namespace this might be a namespace prefix for
// included objects, for which credentials should be requested.
type UsageContext interface {
	String() string
}

type StringUsageContext string

func (s StringUsageContext) String() string {
	return string(s)
}

// ConsumerIdentityProvider is an interface for objects requiring
// credentials, which want to expose the ConsumerId they are
// using to request implicit credentials.
type ConsumerIdentityProvider interface {
	// GetConsumerId provides information about the consumer id
	// used for the object implementing this interface.
	// Optionally a sub context can be given to specify
	// a dedicated type specific sub realm.
	GetConsumerId(uctx ...UsageContext) ConsumerIdentity
	// GetIdentityMatcher provides the identity macher type to use
	// to match the consumer identities configured in a credentials
	// context.
	GetIdentityMatcher() string
}

type _consumers struct {
	data map[string]*_consumer
}

func newConsumers() *_consumers {
	return &_consumers{
		data: map[string]*_consumer{},
	}
}

func (c *_consumers) Set(id ConsumerIdentity, pid ProviderIdentity, creds CredentialsSource) {
	c.data[string(id.Key())] = &_consumer{
		providerId:  pid,
		identity:    id,
		credentials: creds,
	}
}

func (p *_consumers) Unregister(pid ProviderIdentity) {
	for n, c := range p.data {
		if c.providerId == pid {
			delete(p.data, n)
		}
	}
}

func (c *_consumers) Get(id ConsumerIdentity) (CredentialsSource, bool) {
	cred, ok := c.data[string(id.Key())]
	if cred != nil {
		return cred.credentials, true
	}
	return nil, ok
}

// Match matches a given request (pattern) against configured
// identities.
func (c *_consumers) Match(pattern ConsumerIdentity, cur ConsumerIdentity, m IdentityMatcher) (CredentialsSource, ConsumerIdentity) {
	var found *_consumer
	for _, s := range c.data {
		if m(pattern, cur, s.identity) {
			found = s
			cur = s.identity
		}
	}
	if found != nil {
		return found.credentials, cur
	}
	return nil, cur
}

type _consumer struct {
	providerId  ProviderIdentity
	identity    ConsumerIdentity
	credentials CredentialsSource
}

func (c *_consumer) GetCredentials() CredentialsSource {
	return c.credentials
}

////////////////////////////////////////////////////////////////////////////////

type consumerPrio struct {
	ConsumerProvider
	priority int
}

func (c *consumerPrio) GetPriority() int {
	return c.priority
}

func WithPriority(p ConsumerProvider, prio int) ConsumerProvider {
	return &consumerPrio{
		p,
		prio,
	}
}

////////////////////////////////////////////////////////////////////////////////

type PriorityProvider interface {
	GetPriority() int
}

func priority(p interface{}) int {
	if pp, ok := p.(PriorityProvider); ok {
		return pp.GetPriority()
	}
	return 10
}

type consumerProviderRegistry struct {
	lock      sync.RWMutex
	explicit  *_consumers
	providers map[ProviderIdentity]ConsumerProvider
	ordered   []ConsumerProvider
}

func newConsumerProviderRegistry() *consumerProviderRegistry {
	return &consumerProviderRegistry{
		explicit:  newConsumers(),
		providers: map[ProviderIdentity]ConsumerProvider{},
		ordered:   nil,
	}
}

var _ ConsumerProvider = (*consumerProviderRegistry)(nil)

func (p *consumerProviderRegistry) Register(id ProviderIdentity, c ConsumerProvider) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.unregister(id)
	p.providers[id] = c
	p.ordered = generics.MapValues(p.providers)
	sort.Slice(p.ordered, func(a, b int) bool {
		return priority(p.ordered[a]) < priority(p.ordered[b])
	})
}

func (p *consumerProviderRegistry) Unregister(id ProviderIdentity) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.unregister(id)
}

func (p *consumerProviderRegistry) unregister(id ProviderIdentity) {
	p.explicit.Unregister(id)
	if _, ok := p.providers[id]; ok {
		delete(p.providers, id)
		p.ordered = generics.MapValues(p.providers)
		sort.Slice(p.ordered, func(a, b int) bool {
			return priority(p.ordered[a]) < priority(p.ordered[b])
		})
	} else {
		for _, sub := range p.providers {
			sub.Unregister(id)
		}
	}
}

func (p *consumerProviderRegistry) Get(id ConsumerIdentity) (CredentialsSource, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	credsrc, ok := p.explicit.Get(id)
	if ok {
		return credsrc, ok
	}
	for _, sub := range p.providers {
		credsrc, ok := sub.Get(id)
		if ok {
			return credsrc, ok
		}
	}
	return nil, false
}

func (p *consumerProviderRegistry) Match(pattern ConsumerIdentity, cur ConsumerIdentity, m IdentityMatcher) (CredentialsSource, ConsumerIdentity) {
	p.lock.Lock()
	defer p.lock.Unlock()

	credsrc, cur := p.explicit.Match(pattern, cur, m)
	for _, sub := range p.providers {
		var f CredentialsSource
		f, cur = sub.Match(pattern, cur, m)
		if f != nil {
			credsrc = f
		}
	}
	return credsrc, cur
}

func (p *consumerProviderRegistry) Set(id ConsumerIdentity, pid ProviderIdentity, creds CredentialsSource) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.explicit.Set(id, pid, creds)
}
