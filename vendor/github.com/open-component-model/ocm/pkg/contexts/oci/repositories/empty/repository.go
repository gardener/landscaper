// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package empty

import (
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Repository struct{}

var _ cpi.Repository = &Repository{}

func newRepository(ctx datacontext.Context) interface{} {
	return &Repository{}
}

func (r Repository) GetSpecification() cpi.RepositorySpec {
	return NewRepositorySpec()
}

func (r *Repository) NamespaceLister() cpi.NamespaceLister {
	return r
}

func (r *Repository) NumNamespaces(prefix string) (int, error) {
	return 0, nil
}

func (r *Repository) GetNamespaces(prefix string, closure bool) ([]string, error) {
	return nil, nil
}

func (r Repository) ExistsArtifact(name string, version string) (bool, error) {
	return false, nil
}

func (r Repository) LookupArtifact(name string, version string) (cpi.ArtifactAccess, error) {
	return nil, cpi.ErrUnknownArtifact(name, version)
}

func (r Repository) LookupNamespace(name string) (cpi.NamespaceAccess, error) {
	return nil, errors.ErrNotSupported("write access")
}

func (r Repository) Close() error {
	return nil
}
