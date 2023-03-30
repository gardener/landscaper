// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dockerconfig

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockercred "github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Repository struct {
	lock      sync.RWMutex
	ctx       cpi.Context
	propagate bool
	path      string
	config    *configfile.ConfigFile
}

func NewRepository(ctx cpi.Context, path string, propagate bool) (*Repository, error) {
	r := &Repository{
		ctx:       ctx,
		propagate: propagate,
		path:      path,
	}
	err := r.Read(true)
	return r, err
}

var _ cpi.Repository = &Repository{}

func (r *Repository) ExistsCredentials(name string) (bool, error) {
	err := r.Read(false)
	if err != nil {
		return false, err
	}
	r.lock.RLock()
	defer r.lock.RUnlock()

	_, err = r.config.GetAuthConfig(name)
	return err != nil, err
}

func (r *Repository) LookupCredentials(name string) (cpi.Credentials, error) {
	err := r.Read(false)
	if err != nil {
		return nil, err
	}
	r.lock.RLock()
	defer r.lock.RUnlock()

	auth, err := r.config.GetAuthConfig(name)
	if err != nil {
		return nil, err
	}
	return newCredentials(auth), nil
}

func (r *Repository) WriteCredentials(name string, creds cpi.Credentials) (cpi.Credentials, error) {
	return nil, errors.ErrNotSupported("write", "credentials", Type)
}

func (r *Repository) Read(force bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if !force && r.config != nil {
		return nil
	}
	path := r.path
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		path = home + path[1:]
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	cfg, err := config.LoadFromReader(bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	log := r.ctx.Logger()
	defaultStore := dockercred.DetectDefaultStore(cfg.CredentialsStore)
	store := dockercred.NewNativeStore(cfg, defaultStore)
	// get default native credential store
	if r.propagate {
		all := cfg.GetAuthConfigs()
		for h, a := range all {
			hostname := dockercred.ConvertToHostname(h)
			if hostname == "index.docker.io" {
				hostname = "docker.io"
			}
			id := cpi.ConsumerIdentity{
				cpi.ATTR_TYPE:        identity.CONSUMER_TYPE,
				identity.ID_HOSTNAME: hostname,
			}

			var creds cpi.Credentials
			if IsEmptyAuthConfig(a) {
				log.Debug("propagate id with default store", "id", id, "store", defaultStore)

				creds = NewCredentials(r, h, store)
			} else {
				log.Debug("propagate id", "id", id)

				creds = newCredentials(a)
			}
			r.ctx.SetCredentialsForConsumer(id, creds)
		}
		for h, helper := range cfg.CredentialHelpers {
			hostname := dockercred.ConvertToHostname(h)
			if hostname == "index.docker.io" {
				hostname = "docker.io"
			}
			id := cpi.ConsumerIdentity{
				cpi.ATTR_TYPE:        identity.CONSUMER_TYPE,
				identity.ID_HOSTNAME: hostname,
			}

			log.Debug("propagate id with helper", "id", id, "helper", helper)

			r.ctx.SetCredentialsForConsumer(id, NewCredentials(r, h, dockercred.NewNativeStore(cfg, helper)))
		}
	}
	r.config = cfg
	return nil
}

func newCredentials(auth types.AuthConfig) cpi.Credentials {
	props := common.Properties{
		cpi.ATTR_USERNAME: norm(auth.Username),
		cpi.ATTR_PASSWORD: norm(auth.Password),
	}
	props.SetNonEmptyValue("auth", auth.Auth)
	props.SetNonEmptyValue(cpi.ATTR_SERVER_ADDRESS, norm(auth.ServerAddress))
	props.SetNonEmptyValue(cpi.ATTR_IDENTITY_TOKEN, norm(auth.IdentityToken))
	props.SetNonEmptyValue(cpi.ATTR_REGISTRY_TOKEN, norm(auth.RegistryToken))
	return cpi.NewCredentials(props)
}

func norm(s string) string {
	for strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
	}
	return s
}

// IsEmptyAuthConfig validates if the resulting auth config contains credentials.
func IsEmptyAuthConfig(auth types.AuthConfig) bool {
	if len(auth.Auth) != 0 {
		return false
	}
	if len(auth.Username) != 0 {
		return false
	}
	return true
}
