// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	dockerconfig "github.com/docker/cli/cli/config"
	dockercreds "github.com/docker/cli/cli/config/credentials"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
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

// CreateOCIRegistryKeyringFromFilesystem creates a new OCI registry keyring from a given file system.
func CreateOCIRegistryKeyringFromFilesystem(pullSecrets []corev1.Secret, configFiles []string, fs vfs.FileSystem) (OCIKeyring, error) {
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
		dockerConfigBytes, err := vfs.ReadFile(fs, configFile)
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

// CreateOCIRegistryKeyring creates a new OCI registry keyring.
func CreateOCIRegistryKeyring(pullSecrets []corev1.Secret, configFiles []string) (OCIKeyring, error) {
	return CreateOCIRegistryKeyringFromFilesystem(pullSecrets, configFiles, osfs.New())
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
