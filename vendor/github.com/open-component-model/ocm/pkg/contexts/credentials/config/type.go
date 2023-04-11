// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "credentials" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(ConfigType, cfgcpi.NewConfigType(ConfigType, &Config{}, usage))
	cfgcpi.RegisterConfigType(ConfigTypeV1, cfgcpi.NewConfigType(ConfigTypeV1, &Config{}, usage))
}

// Config describes a configuration for the config context.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	// Consumers describe predefine logical cosumer specs mapped to credentials
	// These will (potentially) be evaluated if access objects requiring credentials
	// are provided by other modules (e.g. oci repo access) without
	// specifying crednentials. Then this module can request credentials here by passing
	// an appropriate consumer spec.
	Consumers []ConsumerSpec `json:"consumers,omitempty"`
	// Repositories describe preloaded credential repositories with potential credential chain
	Repositories []RepositorySpec `json:"repositories,omitempty"`
	// Aliases describe logical credential repositories mapped to implementing repositories
	Aliases map[string]RepositorySpec `json:"aliases,omitempty"`
}

type ConsumerSpec struct {
	Identity    cpi.ConsumerIdentity         `json:"identity"`
	Credentials []cpi.GenericCredentialsSpec `json:"credentials"`
}

type RepositorySpec struct {
	Repository  cpi.GenericRepositorySpec    `json:"repository"`
	Credentials []cpi.GenericCredentialsSpec `json:"credentials,omitempty"`
}

// NewConfigSpec creates a new memory ConfigSpec.
func New() *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedObjectType(ConfigType),
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) MapCredentialsChain(creds ...cpi.CredentialsSpec) ([]cpi.GenericCredentialsSpec, error) {
	var cgens []cpi.GenericCredentialsSpec
	for _, c := range creds {
		cgen, err := cpi.ToGenericCredentialsSpec(c)
		if err != nil {
			return nil, err
		}
		cgens = append(cgens, *cgen)
	}
	return cgens, nil
}

func (a *Config) AddConsumer(id cpi.ConsumerIdentity, creds ...cpi.CredentialsSpec) error {
	cgens, err := a.MapCredentialsChain(creds...)
	if err != nil {
		return fmt.Errorf("failed to map credentials chain: %w", err)
	}

	spec := &ConsumerSpec{
		Identity:    id,
		Credentials: cgens,
	}
	a.Consumers = append(a.Consumers, *spec)
	return nil
}

func (a *Config) MapRepository(repo cpi.RepositorySpec, creds ...cpi.CredentialsSpec) (*RepositorySpec, error) {
	rgen, err := cpi.ToGenericRepositorySpec(repo)
	if err != nil {
		return nil, err
	}

	cgens, err := a.MapCredentialsChain(creds...)
	if err != nil {
		return nil, err
	}

	return &RepositorySpec{
		Repository:  *rgen,
		Credentials: cgens,
	}, nil
}

func (a *Config) AddRepository(repo cpi.RepositorySpec, creds ...cpi.CredentialsSpec) error {
	spec, err := a.MapRepository(repo, creds...)
	if err != nil {
		return fmt.Errorf("failed to map repository: %w", err)
	}

	a.Repositories = append(a.Repositories, *spec)

	return nil
}

func (a *Config) AddAlias(name string, repo cpi.RepositorySpec, creds ...cpi.CredentialsSpec) error {
	spec, err := a.MapRepository(repo, creds...)
	if err != nil {
		return fmt.Errorf("failed to map repository: %w", err)
	}

	if a.Aliases == nil {
		a.Aliases = map[string]RepositorySpec{}
	}
	a.Aliases[name] = *spec
	return nil
}

func (a *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	list := errors.ErrListf("applying config")
	t, ok := target.(cpi.Context)
	if !ok {
		return cfgcpi.ErrNoContext(ConfigType)
	}
	for _, e := range a.Consumers {
		t.SetCredentialsForConsumer(e.Identity, CredentialsChain(e.Credentials...))
	}
	sub := errors.ErrListf("applying aliases")
	for n, e := range a.Aliases {
		sub.Add(t.SetAlias(n, &e.Repository, CredentialsChain(e.Credentials...)))
	}
	list.Add(sub.Result())
	sub = errors.ErrListf("applying repositories")
	for i, e := range a.Repositories {
		_, err := t.RepositoryForSpec(&e.Repository, CredentialsChain(e.Credentials...))
		sub.Add(errors.Wrapf(err, "repository entry %d", i))
	}
	list.Add(sub.Result())

	return list.Result()
}

func CredentialsChain(creds ...cpi.GenericCredentialsSpec) cpi.CredentialsChain {
	r := make([]cpi.CredentialsSource, len(creds))
	for i := range creds {
		r[i] = &creds[i]
	}
	return r
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define a list
of arbitrary configuration specifications:

<pre>
    type: ` + ConfigType + `
    consumers:
      - identity:
          &lt;name>: &lt;value>
          ...
        credentials:
          - &lt;credential specification>
          ... credential chain
    repositories:
       - repository: &lt;repository specification>
         credentials:
          - &lt;credential specification>
          ... credential chain
    aliases:
       &lt;name>: 
         repository: &lt;repository specification>
         credentials:
          - &lt;credential specification>
          ... credential chain
</pre>
`
