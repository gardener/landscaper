// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/helm/identity"
)

type Option interface {
	apply(dl *chartDownloader) error
}

////////////////////////////////////////////////////////////////////////////////

type credOption struct {
	creds common.Properties
}

func (c *credOption) apply(dl *chartDownloader) error {
	if c.creds != nil {
		dl.creds = c.creds
	}
	return nil
}

func WithCredentials(creds common.Properties) Option {
	return &credOption{creds}
}

////////////////////////////////////////////////////////////////////////////////

type authOption struct {
	user, password string
}

func (c *authOption) apply(dl *chartDownloader) error {
	if dl.creds == nil {
		dl.creds = common.Properties{}
	}
	dl.creds[identity.ATTR_USERNAME] = c.user
	dl.creds[identity.ATTR_PASSWORD] = c.password
	return nil
}

func WithBasicAuth(user, password string) Option {
	return &authOption{user, password}
}

////////////////////////////////////////////////////////////////////////////////

type certOption struct {
	cert    []byte
	privkey []byte
}

func (c *certOption) apply(dl *chartDownloader) error {
	if len(c.privkey) != 0 {
		if dl.creds == nil {
			dl.creds = common.Properties{}
		}
		dl.creds[identity.ATTR_CERTIFICATE] = string(c.cert)
		dl.creds[identity.ATTR_PRIVATE_KEY] = string(c.privkey)
	}
	return nil
}

func WithCert(cert []byte, privkey []byte) Option {
	return &certOption{cert, privkey}
}

////////////////////////////////////////////////////////////////////////////////

type cacertOption struct {
	data []byte
}

func (c *cacertOption) apply(dl *chartDownloader) error {
	if len(c.data) > 0 {
		if dl.creds == nil {
			dl.creds = common.Properties{}
		}
		dl.creds[identity.ATTR_CERTIFICATE_AUTHORITY] = string(c.data)
	}
	return nil
}

func WithRootCert(data []byte) Option {
	return &cacertOption{data}
}

////////////////////////////////////////////////////////////////////////////////

type keyringOption struct {
	data []byte
}

func (c *keyringOption) apply(dl *chartDownloader) error {
	if len(c.data) > 0 {
		dl.keyring = c.data
	}
	return nil
}

func WithKeyring(data []byte) Option {
	return &keyringOption{data}
}
