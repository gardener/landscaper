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
	"github.com/docker/cli/cli/config/types"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
)

type Repository struct {
	lock      sync.RWMutex
	ctx       cpi.Context
	propagate bool
	path      string
	data      []byte
	config    *configfile.ConfigFile
}

func NewRepository(ctx cpi.Context, path string, data []byte, propagate bool) (*Repository, error) {
	r := &Repository{
		ctx:       ctx,
		propagate: propagate,
		path:      path,
		data:      data,
	}
	if path != "" && len(data) > 0 {
		return nil, fmt.Errorf("only config file or config data possible")
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
	var (
		data []byte
		err  error
		id   finalizer.ObjectIdentity
	)
	if r.path != "" {
		path := r.path
		if strings.HasPrefix(path, "~/") {
			home := os.Getenv("HOME")
			path = home + path[1:]
		}

		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", path, err)
		}
		id = cpi.ProviderIdentity(PROVIDER + "/" + path)
	} else if len(r.data) > 0 {
		data = r.data
		id = finalizer.NewObjectIdentity(PROVIDER)
	}

	cfg, err := config.LoadFromReader(bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if r.propagate {
		r.ctx.RegisterConsumerProvider(id, &ConsumerProvider{cfg})
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
