// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	dockerconfig "github.com/docker/cli/cli/config"
	dockercreds "github.com/docker/cli/cli/config/credentials"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	corev1 "k8s.io/api/core/v1"
)

// OCIKeyring is the interface that implements are keyring to retrieve credentials for a given
// server.
type OCIKeyring interface {
	// Get retrieves credentials from the keyring for a given resource url.
	Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool)
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context, client *http.Client, plainHTTP bool) (remotes.Resolver, error)
}

// CreateOCIRegistryKeyring creates a new OCI registry keyring.
func CreateOCIRegistryKeyring(pullSecrets []corev1.Secret, configFiles []string) (OCIKeyring, error) {
	store := &ociKeyring{
		index: make([]string, 0),
		store: map[string]dockerconfigtypes.AuthConfig{},
	}
	for _, secret := range pullSecrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}

		dockerConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(dockerConfigBytes))
		if err != nil {
			return nil, err
		}

		// currently only support the default credential store.
		credStore := dockerConfig.GetCredentialsStore("")
		if err := store.Add(credStore); err != nil {
			return nil, err
		}
	}

	for _, configFile := range configFiles {
		dockerConfigBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		dockerConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(dockerConfigBytes))
		if err != nil {
			return nil, err
		}

		// currently only support the default credential store.
		credStore := dockerConfig.GetCredentialsStore("")
		if err := store.Add(credStore); err != nil {
			return nil, err
		}
	}

	return store, nil
}

type ociKeyring struct {
	index []string
	store map[string]dockerconfigtypes.AuthConfig
}

var _ OCIKeyring = &ociKeyring{}

func (o ociKeyring) Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool) {
	// todo: check how to include default docker registry
	for _, u := range o.index {
		if strings.HasPrefix(u, resourceURl) {
			return o.store[u], true
		}
	}

	return dockerconfigtypes.AuthConfig{}, false
}

// getCredentials returns the username and password for a given url.
// It implements the Credentials func for a docker resolver
func (o *ociKeyring) getCredentials(url string) (string, string, error) {
	auth, ok := o.Get(url)
	if !ok {
		return "", "", fmt.Errorf("authentication for %s cannot be found", url)
	}

	return auth.Username, auth.Password, nil
}

func (o *ociKeyring) Add(store dockercreds.Store) error {
	auths, err := store.GetAll()
	if err != nil {
		return err
	}
	for address, auth := range auths {
		o.store[address] = auth
		o.index = append(o.index, address)
	}
	return nil
}

func (o *ociKeyring) Resolver(ctx context.Context, client *http.Client, plainHTTP bool) (remotes.Resolver, error) {
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: o.getCredentials,
		Client:      client,
		PlainHTTP:   plainHTTP,
	}), nil
}
