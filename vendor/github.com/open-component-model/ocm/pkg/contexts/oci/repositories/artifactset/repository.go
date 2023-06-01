// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Repository struct {
	ctx  cpi.Context
	spec *RepositorySpec
	arch *ArtifactSet
}

var _ cpi.Repository = (*Repository)(nil)

func NewRepository(ctx cpi.Context, s *RepositorySpec) (*Repository, error) {
	if s.PathFileSystem == nil {
		s.PathFileSystem = vfsattr.Get(ctx)
	}
	r := &Repository{ctx, s, nil}
	_, err := r.Open()
	if err != nil {
		return nil, err
	}
	return r, err
}

func (r *Repository) Get() *ArtifactSet {
	if r.arch != nil {
		return r.arch
	}
	return nil
}

func (r *Repository) Open() (*ArtifactSet, error) {
	a, err := Open(r.spec.AccessMode, r.spec.FilePath, 0o700, &Options{}, &r.spec.Options, accessio.PathFileSystem(r.spec.PathFileSystem))
	if err != nil {
		return nil, err
	}
	r.arch = a
	return a, nil
}

func (r *Repository) GetContext() cpi.Context {
	return r.ctx
}

func (r *Repository) GetSpecification() cpi.RepositorySpec {
	return r.spec
}

func (r *Repository) NamespaceLister() cpi.NamespaceLister {
	return anonymous
}

func (r *Repository) ExistsArtifact(name string, ref string) (bool, error) {
	if name != "" {
		return false, nil
	}
	return r.arch.HasArtifact(ref)
}

func (r *Repository) LookupArtifact(name string, ref string) (cpi.ArtifactAccess, error) {
	if name != "" {
		return nil, cpi.ErrUnknownArtifact(name, ref)
	}
	return r.arch.GetArtifact(ref)
}

func (r *Repository) LookupNamespace(name string) (cpi.NamespaceAccess, error) {
	if name != "" {
		return nil, errors.ErrNotSupported("namespace", name)
	}
	return r.arch, nil
}

func (r Repository) Close() error {
	if r.arch != nil {
		r.arch.Close()
	}
	return nil
}

// NamespaceLister handles the namespaces provided by an artifact set.
// This is always single anonymous namespace, which by ddefinition
// is the empty string.
type NamespaceLister struct{}

var anonymous cpi.NamespaceLister = &NamespaceLister{}

// NumNamespaces returns the number of namespaces with a given prefix
// for an artifact set. This is either one (the anonymous namespace) if
// the prefix is empty (all namespaces) or zero if a prefix is given.
func (n *NamespaceLister) NumNamespaces(prefix string) (int, error) {
	if prefix == "" {
		return 1, nil
	}
	return 0, nil
}

// GetNamespaces returns namespaces with a given prefix.
// This is the anonymous namespace ("") for an empty prefix
// or no namespace at all if a prefix is given.
func (n *NamespaceLister) GetNamespaces(prefix string, closure bool) ([]string, error) {
	if prefix == "" {
		return []string{""}, nil
	}
	return nil, nil
}
