// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"
	"net/url"
	"path"
	"strings"

	dockerreference "github.com/containerd/containerd/reference/docker"
	dockercreds "github.com/docker/cli/cli/config/credentials"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/google/go-containerregistry/pkg/authn"

	"github.com/gardener/component-cli/pkg/logcontext"
)

// to find a suitable secret for images on Docker Hub, we need its two domains to do matching
const (
	dockerHubDomain       = "docker.io"
	dockerHubLegacyDomain = "index.docker.io"
)

// UsedUserLogKey describes the key that is injected into the logging context values.
const UsedUserLogKey = "ociUser"

// Auth describes a interface of the dockerconfigtypes.Auth struct
type Auth interface {
	GetUsername() string
	GetPassword() string
	GetAuth() string

	// GetIdentityToken is used to authenticate the user and get
	// an access token for the registry.
	GetIdentityToken() string

	// GetRegistryToken is a bearer token to be sent to a registry
	GetRegistryToken() string
}

// Informer describes a interface that returns optional metadata.
// The Auth interface can be enhanced using metadata
type Informer interface {
	Info() map[string]string
}

// AuthConfig implements the Auth using the docker authconfig type.
// It also implements the Informer interface for additional information
type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`

	// Email is an optional value associated with the username.
	// This field is deprecated and will be removed in a later
	// version of docker.
	Email string `json:"email,omitempty"`

	ServerAddress string `json:"serveraddress,omitempty"`

	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identitytoken,omitempty"`

	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registrytoken,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// FromAuthConfig creates a Auth object using the docker authConfig type
func FromAuthConfig(cfg dockerconfigtypes.AuthConfig, keysAndValues ...string) AuthConfig {
	metadata := make(map[string]string)
	var prevKey string
	for _, v := range keysAndValues {
		if len(prevKey) == 0 {
			prevKey = v
			continue
		}
		metadata[prevKey] = v
		prevKey = ""
	}

	return AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
		Metadata:      metadata,
	}
}

func (a AuthConfig) GetUsername() string {
	return a.Username
}

func (a AuthConfig) GetPassword() string {
	return a.Password
}

func (a AuthConfig) GetAuth() string {
	return a.Auth
}

func (a AuthConfig) GetIdentityToken() string {
	return a.IdentityToken
}

func (a AuthConfig) GetRegistryToken() string {
	return a.RegistryToken
}

func (a AuthConfig) Info() map[string]string {
	return a.Metadata
}

// Keyring enhances the google go-lib auth keyring with a contextified resolver
type Keyring interface {
	authn.Keychain
	// ResolveWithContext looks up the most appropriate credential for the specified target.
	ResolveWithContext(context.Context, authn.Resource) (authn.Authenticator, error)
}

// OCIKeyring is the interface that implements are keyring to retrieve credentials for a given
// server.
type OCIKeyring interface {
	authn.Keychain
	// ResolveWithContext looks up the most appropriate credential for the specified target.
	ResolveWithContext(context.Context, authn.Resource) (authn.Authenticator, error)
	// Get retrieves credentials from the keyring for a given resource url.
	Get(resourceURl string) Auth
	// GetCredentials returns the username and password for a hostname if defined.
	GetCredentials(hostname string) (username, password string, err error)
}

// AuthConfigGetter is a function that returns a auth config for a given host name
type AuthConfigGetter func(address string) (Auth, error)

// DefaultAuthConfigGetter describes a default getter method for a authentication method
func DefaultAuthConfigGetter(config Auth) AuthConfigGetter {
	return func(_ string) (Auth, error) {
		return config, nil
	}
}

// GeneralOciKeyring is general implementation of a oci keyring that can be extended with other credentials.
type GeneralOciKeyring struct {
	// index is an additional index structure that also contains multi
	index *IndexNode
	store map[string][]AuthConfigGetter
}

type IndexNode struct {
	Segment   string
	Addresses []string
	Children  []*IndexNode
}

func (n *IndexNode) Set(path string, addresses ...string) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		n.Addresses = append(n.Addresses, addresses...)
		return
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		child = &IndexNode{
			Segment: splitPath[0],
		}
		n.Children = append(n.Children, child)
	}
	child.Set(strings.Join(splitPath[1:], "/"), addresses...)
}

func (n *IndexNode) FindSegment(segment string) *IndexNode {
	for _, child := range n.Children {
		if child.Segment == segment {
			return child
		}
	}
	return nil
}

func (n *IndexNode) Find(path string) ([]string, bool) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		return n.Addresses, true
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		// returns the current address if no more specific auth config is defined
		return n.Addresses, true
	}
	return child.Find(strings.Join(splitPath[1:], "/"))
}

// New creates a new empty general oci keyring.
func New() *GeneralOciKeyring {
	return &GeneralOciKeyring{
		index: &IndexNode{},
		store: make(map[string][]AuthConfigGetter),
	}
}

var _ OCIKeyring = &GeneralOciKeyring{}

// Size returns the size of the keyring
func (o GeneralOciKeyring) Size() int {
	return len(o.store)
}

func (o GeneralOciKeyring) Get(resourceURl string) Auth {
	ref, err := dockerreference.ParseDockerRef(resourceURl)
	if err == nil {
		// if the name is not conical try to treat it like a host name
		resourceURl = ref.Name()
	}
	if auth := o.get(resourceURl); auth != nil {
		return auth
	}

	// fallback to legacy docker domain if applicable
	// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
	if ref != nil && dockerreference.Domain(ref) == dockerHubDomain {
		dockerreference.Path(ref)
		return o.get(path.Join(dockerHubLegacyDomain, dockerreference.Path(ref)))
	}
	return nil
}

func (o GeneralOciKeyring) get(url string) Auth {
	addresses, ok := o.index.Find(url)
	if !ok {
		return nil
	}
	for _, address := range addresses {
		authGetters, ok := o.store[address]
		if !ok {
			continue
		}
		for _, authGetter := range authGetters {
			auth, err := authGetter(url)
			if err != nil {
				// todo: add logger
				continue
			}
			if IsEmptyAuthConfig(auth) {
				// try another config if the current one is emtpy
				continue
			}
			return auth
		}

	}
	return nil
}

// GetCredentials returns the username and password for a given hostname.
// It implements the Credentials func for a docker resolver
func (o *GeneralOciKeyring) GetCredentials(hostname string) (username, password string, err error) {
	auth := o.get(hostname)
	if auth == nil {
		// fallback to legacy docker domain if applicable
		// this is how containerd translates the old domain for DockerHub to the new one, taken from containerd/reference/docker/reference.go:674
		if hostname == dockerHubDomain {
			return o.GetCredentials(dockerHubLegacyDomain)
		}
		// try authentication with no username and password.
		// this is needed by some registries like ghcr that require a auth token flow even for public images.
		return "", "", nil

		// todo: add log for the error if now authentication can be found
		//return "", "", fmt.Errorf("authentication for %s cannot be found", hostname)
	}

	return auth.GetUsername(), auth.GetPassword(), nil
}

// AddAuthConfig adds a auth config for a address
func (o *GeneralOciKeyring) AddAuthConfig(address string, auth Auth) error {
	return o.AddAuthConfigGetter(address, DefaultAuthConfigGetter(auth))
}

// AddAuthConfigGetter adds a auth config for a address
func (o *GeneralOciKeyring) AddAuthConfigGetter(address string, getter AuthConfigGetter) error {
	// normalize host name
	var err error
	address, err = normalizeHost(address)
	if err != nil {
		return err
	}
	o.store[address] = append(o.store[address], getter)
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
		if err := o.AddAuthConfig(address, FromAuthConfig(auth)); err != nil {
			return err
		}
	}
	return nil
}

// Resolve implements the google container registry auth interface.
func (o *GeneralOciKeyring) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	return o.ResolveWithContext(context.TODO(), resource)
}

// ResolveWithContext implements the google container registry auth interface.
func (o *GeneralOciKeyring) ResolveWithContext(ctx context.Context, resource authn.Resource) (authn.Authenticator, error) {
	authconfig := o.Get(resource.String())
	if authconfig == nil {
		logcontext.AddContextValue(ctx, UsedUserLogKey, "anonymous")
		return authn.FromConfig(authn.AuthConfig{}), nil
	}

	if ctxVal := logcontext.FromContext(ctx); ctxVal != nil {
		(*ctxVal)[UsedUserLogKey] = authconfig.GetUsername()
		if informer, ok := authconfig.(Informer); ok {
			ctxVal := logcontext.FromContext(ctx)
			for key, val := range informer.Info() {
				(*ctxVal)[key] = val
			}
		}
	}

	return authn.FromConfig(authn.AuthConfig{
		Username:      authconfig.GetUsername(),
		Password:      authconfig.GetPassword(),
		Auth:          authconfig.GetAuth(),
		IdentityToken: authconfig.GetIdentityToken(),
		RegistryToken: authconfig.GetRegistryToken(),
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

// Merge merges all authentication options from keyring 1 and 2.
// Keyring 2 overwrites authentication from keyring 1 on clashes.
func Merge(k1, k2 *GeneralOciKeyring) error {
	for address, getters := range k2.store {
		for _, getter := range getters {
			if err := k1.AddAuthConfigGetter(address, getter); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsEmptyAuthConfig validates if the resulting auth config contains credentails
func IsEmptyAuthConfig(auth Auth) bool {
	if len(auth.GetAuth()) != 0 {
		return false
	}
	if len(auth.GetUsername()) != 0 {
		return false
	}
	return true
}
