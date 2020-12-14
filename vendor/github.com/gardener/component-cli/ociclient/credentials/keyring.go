// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	dockerreference "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	dockerconfig "github.com/docker/cli/cli/config"
	dockercreds "github.com/docker/cli/cli/config/credentials"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	corev1 "k8s.io/api/core/v1"
)

// to find a suitable secret for images on Docker Hub, we need its two domains to do matching
const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"
)

// OCIKeyring is the interface that implements are keyring to retrieve credentials for a given
// server.
type OCIKeyring interface {
	// Get retrieves credentials from the keyring for a given resource url.
	Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool)
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error)
}

// CreateOCIRegistryKeyringFromFilesystem creates a new OCI registry keyring from a given file system.
func CreateOCIRegistryKeyringFromFilesystem(pullSecrets []corev1.Secret, configFiles []string, fs vfs.FileSystem) (*GeneralOciKeyring, error) {
	store := &GeneralOciKeyring{
		index: &IndexNode{},
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
func CreateOCIRegistryKeyring(pullSecrets []corev1.Secret, configFiles []string) (*GeneralOciKeyring, error) {
	return CreateOCIRegistryKeyringFromFilesystem(pullSecrets, configFiles, osfs.New())
}

// GeneralOciKeyring is general implementation of a oci keyring that can be extended with other credentials.
type GeneralOciKeyring struct {
	// index is an additional index structure that also contains multi
	index *IndexNode
	store map[string]dockerconfigtypes.AuthConfig
}

type IndexNode struct {
	Segment  string
	Address  string
	Children []*IndexNode
}

func (n *IndexNode) Set(path, address string) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		n.Address = address
		return
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		child = &IndexNode{
			Segment: splitPath[0],
		}
		n.Children = append(n.Children, child)
	}
	child.Set(strings.Join(splitPath[1:], "/"), address)
}

func (n *IndexNode) FindSegment(segment string) *IndexNode {
	for _, child := range n.Children {
		if child.Segment == segment {
			return child
		}
	}
	return nil
}

func (n *IndexNode) Find(path string) (string, bool) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		return n.Address, true
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		// returns the current address if no more specific auth config is defined
		return n.Address, true
	}
	return child.Find(strings.Join(splitPath[1:], "/"))
}

// New creates a new empty general oci keyring.
func New() *GeneralOciKeyring {
	return &GeneralOciKeyring{
		index: &IndexNode{},
		store: make(map[string]dockerconfigtypes.AuthConfig),
	}
}

var _ OCIKeyring = &GeneralOciKeyring{}

// Size returns the size of the keyring
func (o GeneralOciKeyring) Size() int {
	return len(o.store)
}

func (o GeneralOciKeyring) Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool) {
	ref, err := dockerreference.ParseDockerRef(resourceURl)
	if err == nil {
		// if the name is not conical try to treat it like a host name
		resourceURl = ref.Name()
	}
	if auth, ok := o.get(resourceURl); ok {
		return auth, true
	}

	// fallback to legacy docker domain if applicable
	// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
	if dockerreference.Domain(ref) == dockerHubDomain {
		dockerreference.Path(ref)
		return o.get(path.Join(dockerHubLegacyDomain, dockerreference.Path(ref)))
	}
	return dockerconfigtypes.AuthConfig{}, false
}

func (o GeneralOciKeyring) get(url string) (dockerconfigtypes.AuthConfig, bool) {
	address, ok := o.index.Find(url)
	if !ok {
		return dockerconfigtypes.AuthConfig{}, false
	}
	if auth, ok := o.store[address]; ok {
		return auth, ok
	}
	return dockerconfigtypes.AuthConfig{}, false
}

// GetCredentials returns the username and password for a given hostname.
// It implements the Credentials func for a docker resolver
func (o *GeneralOciKeyring) GetCredentials(hostname string) (username, password string, err error) {
	auth, ok := o.get(hostname)
	if !ok {
		// fallback to legacy docker domain if applicable
		// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
		if hostname == dockerHubDomain {
			return o.GetCredentials(dockerHubLegacyDomain)
		}
		return "", "", fmt.Errorf("authentication for %s cannot be found", hostname)
	}

	return auth.Username, auth.Password, nil
}

// AddAuthConfig adds a auth config for a address
func (o *GeneralOciKeyring) AddAuthConfig(address string, auth dockerconfigtypes.AuthConfig) error {
	// normalize host name
	var err error
	address, err = normalizeHost(address)
	if err != nil {
		return err
	}
	o.store[address] = auth
	o.index.Set(address, address)
	return nil
}

// Add adds all addresses of a docker credential store.
func (o *GeneralOciKeyring) Add(store dockercreds.Store) error {
	auths, err := store.GetAll()
	if err != nil {
		return err
	}
	for address, auth := range auths {
		if err := o.AddAuthConfig(address, auth); err != nil {
			return err
		}
	}
	return nil
}

func (o *GeneralOciKeyring) Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error) {
	if ref == "" {
		return docker.NewResolver(docker.ResolverOptions{
			Credentials: o.GetCredentials,
			Client:      client,
			PlainHTTP:   plainHTTP,
		}), nil
	}

	// get specific auth for ref and only return a resolver with that authentication config
	auth, ok := o.Get(ref)
	if !ok {
		return docker.NewResolver(docker.ResolverOptions{
			Credentials: o.GetCredentials,
			Client:      client,
			PlainHTTP:   plainHTTP,
		}), nil
	}
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: func(url string) (string, string, error) {
			return auth.Username, auth.Password, nil
		},
		Client:    client,
		PlainHTTP: plainHTTP,
	}), nil
}

func normalizeHost(u string) (string, error) {
	if !strings.Contains(u, "://") {
		u = "dummy://" + u
	}
	host, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return path.Join(host.Host, host.Path), nil
}
