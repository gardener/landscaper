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

type RepositoryImpl struct {
	cpi.RepositoryImplBase
	spec *RepositorySpec
	arch *ArtifactSet
}

var _ cpi.RepositoryImpl = (*RepositoryImpl)(nil)

func NewRepository(ctx cpi.Context, s *RepositorySpec) (cpi.Repository, error) {
	if s.PathFileSystem == nil {
		s.PathFileSystem = vfsattr.Get(ctx)
	}
	r := &RepositoryImpl{
		RepositoryImplBase: cpi.NewRepositoryImplBase(ctx),
		spec:               s,
	}
	_, err := r.open()
	if err != nil {
		return nil, err
	}
	return cpi.NewRepository(r, "OCI artifactset"), nil
}

func (r *RepositoryImpl) Get() *ArtifactSet {
	if r.arch != nil {
		return r.arch
	}
	return nil
}

func (r *RepositoryImpl) open() (*ArtifactSet, error) {
	a, err := Open(r.spec.AccessMode, r.spec.FilePath, 0o700, &Options{}, &r.spec.Options, accessio.PathFileSystem(r.spec.PathFileSystem))
	if err != nil {
		return nil, err
	}
	r.arch = a
	return a, nil
}

func (r *RepositoryImpl) GetSpecification() cpi.RepositorySpec {
	return r.spec
}

func (r *RepositoryImpl) NamespaceLister() cpi.NamespaceLister {
	return anonymous
}

func (r *RepositoryImpl) ExistsArtifact(name string, ref string) (bool, error) {
	if name != "" {
		return false, nil
	}
	return r.arch.HasArtifact(ref)
}

func (r *RepositoryImpl) LookupArtifact(name string, ref string) (cpi.ArtifactAccess, error) {
	if name != "" {
		return nil, cpi.ErrUnknownArtifact(name, ref)
	}
	return r.arch.GetArtifact(ref)
}

func (r *RepositoryImpl) LookupNamespace(name string) (cpi.NamespaceAccess, error) {
	if name != "" {
		return nil, errors.ErrNotSupported("namespace", name)
	}
	return r.arch.Dup()
}

func (r RepositoryImpl) Close() error {
	if r.arch != nil {
		return r.arch.Close()
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
