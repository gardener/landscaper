// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"strings"

	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/mime"
)

type Key[K any] interface {
	comparable
	IsValid() bool

	GetMediaType() string
	GetArtifactType() string

	SetArtifact(arttype, medtatype string) K
}

type Registry[H any, K Key[K]] struct {
	mappings map[K][]H
}

func NewRegistry[H any, K Key[K]]() *Registry[H, K] {
	return &Registry[H, K]{
		mappings: map[K][]H{},
	}
}

func (p *Registry[H, K]) lookupMedia(key K) []H {
	lookup := key
	for {
		if h, ok := p.mappings[lookup]; ok {
			return h
		}
		if i := strings.LastIndex(lookup.GetMediaType(), "+"); i > 0 {
			lookup = lookup.SetArtifact(lookup.GetArtifactType(), lookup.GetMediaType()[:i])
		} else {
			break
		}
	}
	return nil
}

func (p *Registry[H, K]) GetHandler(key K) []H {
	r := p.mappings[key]
	if r == nil {
		return nil
	}
	return slices.Clone(r)
}

func (p *Registry[H, K]) LookupHandler(key K) []H {
	h := p.lookupMedia(key)
	if len(h) > 0 {
		return h
	}

	mediatype := key.GetMediaType()
	arttype := key.GetArtifactType()
	if h := p.mappings[key.SetArtifact(arttype, "")]; len(h) > 0 {
		return h
	}
	return p.lookupMedia(key.SetArtifact("", mediatype))
}

func (p *Registry[H, K]) LookupKeys(key K) generics.Set[K] {
	found := generics.Set[K]{}

	if len(p.LookupHandler(key)) > 0 {
		found.Add(key)
	}
	if key.GetArtifactType() == "" {
		for k := range p.mappings {
			if k.GetArtifactType() != "" {
				c := k.SetArtifact(k.GetArtifactType(), key.GetMediaType())
				if !found.Contains(c) && len(p.LookupHandler(c)) > 0 {
					found.Add(c)
				}
			}
		}
	} else {
		for k := range p.mappings {
			if mime.IsMoreGeneral(key.GetMediaType(), k.GetMediaType()) {
				c := k.SetArtifact(key.GetArtifactType(), k.GetMediaType())
				if !found.Contains(c) && len(p.LookupHandler(c)) > 0 {
					found.Add(c)
				}
			}
		}
	}
	return found
}

func (p *Registry[H, K]) Register(key K, h H) {
	p.mappings[key] = append(p.mappings[key], h)
}
