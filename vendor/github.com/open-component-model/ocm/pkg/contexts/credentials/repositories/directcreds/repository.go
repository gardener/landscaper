// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package directcreds

import (
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Repository struct {
	Credentials cpi.Credentials
}

func NewRepository(creds cpi.Credentials) cpi.Repository {
	return &Repository{
		Credentials: creds,
	}
}

func (r *Repository) ExistsCredentials(name string) (bool, error) {
	return name == Type, nil
}

func (r *Repository) LookupCredentials(name string) (cpi.Credentials, error) {
	if name != Type && name != "" {
		return nil, cpi.ErrUnknownCredentials(name)
	}
	return r.Credentials, nil
}

func (r *Repository) WriteCredentials(name string, creds cpi.Credentials) (cpi.Credentials, error) {
	return nil, errors.ErrNotSupported(cpi.KIND_CREDENTIALS, "write", "constant credential")
}

var _ cpi.Repository = &Repository{}
