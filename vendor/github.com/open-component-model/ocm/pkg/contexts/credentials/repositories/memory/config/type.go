// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/common"
	cfgcpi "github.com/open-component-model/ocm/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/memory"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

const (
	ConfigType   = "memory.credentials" + cfgcpi.OCM_CONFIG_TYPE_SUFFIX
	ConfigTypeV1 = ConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigType, usage))
	cfgcpi.RegisterConfigType(cfgcpi.NewConfigType[*Config](ConfigTypeV1, usage))
}

// Config describes a configuration for the config context.
type Config struct {
	runtime.ObjectVersionedType `json:",inline"`
	RepoName                    string            `json:"repoName"`
	Credentials                 []CredentialsSpec `json:"credentials,omitempty"`
}

type CredentialsSpec struct {
	CredentialsName string `json:"credentialsName"`
	// Reference refers to credentials store in some othe repo
	Reference *cpi.GenericCredentialsSpec `json:"reference,omitempty"`
	// Credentials are direct credentials (one of Reference or Credentials must be set)
	Credentials common.Properties `json:"credentials"`
}

// New creates a new memory ConfigSpec.
func New(repo string, credentials ...CredentialsSpec) *Config {
	return &Config{
		ObjectVersionedType: runtime.NewVersionedTypedObject(ConfigType),
		RepoName:            repo,
		Credentials:         credentials,
	}
}

func (a *Config) GetType() string {
	return ConfigType
}

func (a *Config) AddCredentials(name string, props common.Properties) error {
	a.Credentials = append(a.Credentials, CredentialsSpec{CredentialsName: name, Credentials: props})
	return nil
}

func (a *Config) AddCredentialsRef(name string, refname string, spec cpi.RepositorySpec) error {
	repo, err := cpi.ToGenericRepositorySpec(spec)
	if err != nil {
		return fmt.Errorf("unable to convert cpi repository spec to generic: %w", err)
	}

	ref := cpi.NewGenericCredentialsSpec(refname, repo)
	a.Credentials = append(a.Credentials, CredentialsSpec{CredentialsName: name, Reference: ref})

	return nil
}

func (a *Config) ApplyTo(ctx cfgcpi.Context, target interface{}) error {
	list := errors.ErrListf("applying config")

	t, ok := target.(cpi.Context)
	if !ok {
		return cfgcpi.ErrNoContext(ConfigType)
	}

	repo, err := t.RepositoryForSpec(memory.NewRepositorySpec(a.RepoName))
	if err != nil {
		return fmt.Errorf("unable to get repository for spec: %w", err)
	}

	mem, ok := repo.(*memory.Repository)
	if !ok {
		return fmt.Errorf("invalid type assertion of type %T to memory.Repository", repo)
	}

	for i, e := range a.Credentials {
		var creds cpi.Credentials
		if e.Reference != nil {
			if len(e.Credentials) != 0 {
				err = fmt.Errorf("credentials and reference set")
			} else {
				creds, err = e.Reference.Credentials(t)
			}
		} else {
			creds = cpi.NewCredentials(e.Credentials)
		}
		if err != nil {
			list.Add(errors.Wrapf(err, "config entry %d[%s]", i, e.CredentialsName))
		}
		if creds != nil {
			_, err = mem.WriteCredentials(e.CredentialsName, creds)
			if err != nil {
				list.Add(errors.Wrapf(err, "config entry %d", i))
			}
		}
	}
	return list.Result()
}

const usage = `
The config type <code>` + ConfigType + `</code> can be used to define a list
of arbitrary credentials stored in a memory based credentials repository:

<pre>
    type: ` + ConfigType + `
    repoName: default
    credentials:
      - credentialsName: ref
        reference:  # refer to a credential set stored in some other credential repository
          type: Credentials # this is a repo providing just one explicit credential set
          properties:
            username: mandelsoft
            password: specialsecret
      - credentialsName: direct
        credentials: # direct credential specification
            username: mandelsoft2
            password: specialsecret2
</pre>
`
