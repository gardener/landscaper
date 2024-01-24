// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault-client-go"
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/vault/identity"
	"github.com/open-component-model/ocm/pkg/errors"
)

const PROVIDER = "ocm.software/credentialprovider/" + Type

const (
	CUSTOM_SECRETS    = "secrets"
	CUSTOM_CONSUMERID = "consumerId"
)

type mapping struct {
	Id   cpi.ConsumerIdentity
	Name string
}

type ConsumerProvider struct {
	lock        sync.Mutex
	credentials map[string]cpi.DirectCredentials
	repository  *Repository
	creds       cpi.CredentialsSource
	consumer    []*mapping

	updated bool
}

var _ cpi.ConsumerProvider = (*ConsumerProvider)(nil)

func NewConsumerProvider(repo *Repository) (*ConsumerProvider, error) {
	src, err := repo.ctx.GetCredentialsForConsumer(repo.id)
	if err != nil {
		return nil, err
	}
	return &ConsumerProvider{
		creds:       src,
		repository:  repo,
		credentials: map[string]cpi.DirectCredentials{},
	}, nil
}

func (p *ConsumerProvider) update() error {
	var err error

	if p.updated {
		return nil
	}
	p.updated = true

	p.creds, err = p.repository.ctx.GetCredentialsForConsumer(p.repository.id, identity.IdentityMatcher)
	if err != nil {
		return err
	}
	creds, err := p.creds.Credentials(p.repository.ctx)
	if err != nil {
		return err
	}
	err = p.validateCreds(creds)
	if err != nil {
		return err
	}

	p.credentials = map[string]cpi.DirectCredentials{}
	p.consumer = nil

	ctx := context.Background()

	client, err := vault.New(
		vault.WithAddress(p.repository.spec.ServerURL),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		return err
	}

	// vault.WithMountPath("piper/PIPELINE-GROUP-4953/PIPELINE-25042/appRoleCredentials"),
	token, err := p.getToken(ctx, client, creds)
	if err != nil {
		return err
	}

	if err := client.SetToken(token); err != nil {
		return err
	}
	if err := client.SetNamespace(p.repository.spec.Namespace); err != nil {
		return err
	}

	// TODO: support for pure path based access for other secret engine types
	secrets := slices.Clone(p.repository.spec.Secrets)
	if len(secrets) == 0 {
		s, err := client.Secrets.KvV2List(ctx, p.repository.spec.Path,
			vault.WithMountPath(p.repository.spec.SecretsEngine))
		if err != nil {
			return err
		}
		for _, k := range s.Data.Keys {
			if !strings.HasSuffix(k, "/") {
				secrets = append(secrets, k)
			}
		}
	}
	for i := 0; i < len(secrets); i++ {
		n := secrets[i]
		creds, id, list, err := p.read(ctx, client, n)
		p.error(err, "error reading vault secret", n)
		if err == nil {
			for _, a := range list {
				if !slices.Contains(secrets, a) {
					secrets = append(secrets, a)
				}
			}
			if len(id) > 0 {
				p.consumer = append(p.consumer, &mapping{
					Id:   cpi.ConsumerIdentity(id),
					Name: n,
				})
			}
			if len(creds) > 0 {
				p.credentials[n] = cpi.DirectCredentials(creds)
			}
		}
	}
	return nil
}

func (p *ConsumerProvider) validateCreds(creds cpi.Credentials) error {
	m := creds.GetProperty(identity.ATTR_AUTHMETH)
	if m == "" {
		return errors.ErrRequired(identity.ATTR_AUTHMETH)
	}
	meth := methods.Get(m)
	if meth == nil {
		return errors.ErrInvalid(identity.ATTR_AUTHMETH, m)
	}
	return meth.Validate(creds)
}

func (p *ConsumerProvider) getToken(ctx context.Context, client *vault.Client, creds cpi.Credentials) (string, error) {
	m := creds.GetProperty(identity.ATTR_AUTHMETH)
	return methods.Get(m).GetToken(ctx, client, p.repository.spec.Namespace, creds)
}

func (p *ConsumerProvider) error(err error, msg string, secret string, keypairs ...interface{}) {
	if err == nil {
		return
	}
	log.Error(msg, append(keypairs,
		"server", p.repository.spec.ServerURL,
		"namespace", p.repository.spec.Namespace,
		"engine", p.repository.spec.SecretsEngine,
		"path", path.Join(p.repository.spec.Path, secret),
		"error", err.Error(),
	)...,
	)
}

func (p *ConsumerProvider) read(ctx context.Context, client *vault.Client, secret string) (common.Properties, common.Properties, []string, error) {
	// read the secret

	secret = path.Join(p.repository.spec.Path, secret)
	s, err := client.Secrets.KvV2Read(ctx, secret,
		vault.WithMountPath(p.repository.spec.SecretsEngine))
	if err != nil {
		return nil, nil, nil, err
	}

	var id common.Properties
	var list []string
	props := getProps(s.Data.Data)

	if meta, ok := s.Data.Metadata["custom_metadata"].(map[string]interface{}); ok {
		sub := false
		if cid := meta[CUSTOM_CONSUMERID]; cid != nil {
			id = common.Properties{}
			if err := json.Unmarshal([]byte(cid.(string)), &id); err != nil {
				id = nil
			}
			sub = true
		}
		if cid := meta[CUSTOM_SECRETS]; cid != nil {
			if s, ok := meta[CUSTOM_SECRETS].(string); ok {
				for _, e := range strings.Split(s, ",") {
					e = strings.TrimSpace(e)
					if e != "" {
						list = append(list, e)
					}
				}
			}
			sub = true
		}
		if _, ok := meta[cpi.ID_TYPE]; !sub && ok {
			id = getProps(meta)
		}
	}
	return props, id, list, nil
}

func getProps(data map[string]interface{}) common.Properties {
	props := common.Properties{}
	for k, v := range data {
		if s, ok := v.(string); ok {
			props[k] = s
		}
	}
	return props
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ConsumerProvider interface

func (p *ConsumerProvider) Unregister(id cpi.ProviderIdentity) {
}

func (p *ConsumerProvider) Match(req cpi.ConsumerIdentity, cur cpi.ConsumerIdentity, m cpi.IdentityMatcher) (cpi.CredentialsSource, cpi.ConsumerIdentity) {
	return p.get(req, cur, m)
}

func (p *ConsumerProvider) Get(req cpi.ConsumerIdentity) (cpi.CredentialsSource, bool) {
	creds, _ := p.get(req, nil, cpi.CompleteMatch)
	return creds, creds != nil
}

func (p *ConsumerProvider) get(req cpi.ConsumerIdentity, cur cpi.ConsumerIdentity, m cpi.IdentityMatcher) (cpi.CredentialsSource, cpi.ConsumerIdentity) {
	if req.Equals(p.repository.id) {
		return nil, cur
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	p.update()
	var creds cpi.CredentialsSource

	for _, a := range p.consumer {
		if m(req, cur, a.Id) {
			cur = a.Id
			creds = p.credentials[a.Name]
		}
	}
	return creds, cur
}

////////////////////////////////////////////////////////////////////////////////
// lookup

func (c *ConsumerProvider) ExistsCredentials(name string) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := c.update()
	if err != nil {
		return false, err
	}
	_, ok := c.credentials[name]
	return ok, nil
}

func (c *ConsumerProvider) LookupCredentials(name string) (cpi.Credentials, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := c.update()
	if err != nil {
		return nil, err
	}
	src, ok := c.credentials[name]
	if ok {
		return src, nil
	}
	return nil, nil
}
