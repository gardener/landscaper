// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secretserver

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/component-cli/ociclient/credentials"
)

// ContainerRegistryConfigType is the cc secret server container registry config type
const ContainerRegistryConfigType = "container_registry"

// EndpointEnvVarName is the name of the envvar that contains the endpoint of the secret server.
const EndpointEnvVarName = "SECRETS_SERVER_ENDPOINT"

// ConcourseConfigEnvVarName is the name of the envvar that contains the name of concourse config.
const ConcourseConfigEnvVarName = "SECRETS_SERVER_CONCOURSE_CFG_NAME"

// SecretKeyEnvVarName is the name of the envar that contains the decryption key.
const SecretKeyEnvVarName = "SECRET_KEY"

// CipherEnvVarName is the name of the envvar that contains the decryption cipher algorithm.
const CipherEnvVarName = "SECRET_CIPHER_ALGORITHM"

type Privilege string

const (
	ReadOnly  Privilege = "readonly"
	ReadWrite Privilege = "readwrite"
)

// SecretServerConfig is the struct that describes the secret server concourse config
type SecretServerConfig struct {
	ContainerRegistry map[string]*ContainerRegistryCredentials `json:"container_registry"`
}

// ContainerRegistryCredentials describes the container registry credentials struct as given by the cc secrets server.
type ContainerRegistryCredentials struct {
	Username               string    `json:"username"`
	Password               string    `json:"password"`
	Privileges             Privilege `json:"privileges"`
	Host                   string    `json:"host,omitempty"`
	ImageReferencePrefixes []string  `json:"image_reference_prefixes,omitempty"`
}

// KeyringBuilder is a builder that creates a keyring from a concourse config file.
type KeyringBuilder struct {
	log           logr.Logger
	fs            vfs.FileSystem
	path          string
	minPrivileges Privilege
	forRef        string
}

// New creates a new keyring builder.
func New() *KeyringBuilder {
	return &KeyringBuilder{
		log: logr.Discard(),
	}
}

// WithLog configures a optional logger
func (kb *KeyringBuilder) WithLog(log logr.Logger) *KeyringBuilder {
	kb.log = log
	return kb
}

// WithFS configures the builder to use a different filesystem
func (kb *KeyringBuilder) WithFS(fs vfs.FileSystem) *KeyringBuilder {
	kb.fs = fs
	return kb
}

// FromPath configures local concourse config file.
func (kb *KeyringBuilder) FromPath(path string) *KeyringBuilder {
	kb.path = path
	return kb
}

// For configures the builder to only include the config that one reference.
func (kb *KeyringBuilder) For(ref string) *KeyringBuilder {
	kb.forRef = ref
	return kb
}

// WithMinPrivileges configures the builder to only include credentials with a minimal config
func (kb *KeyringBuilder) WithMinPrivileges(priv Privilege) *KeyringBuilder {
	kb.minPrivileges = priv
	return kb
}

// Build creates a oci keyring based on the given configuration.
// It returns nil if now credentials can be found.
func (kb *KeyringBuilder) Build() (*credentials.GeneralOciKeyring, error) {
	keyring := credentials.New()
	if err := kb.Apply(keyring); err != nil {
		return nil, err
	}
	if keyring.Size() == 0 {
		return nil, nil
	}
	return keyring, nil
}

// Apply applies the found configuration to the given keyring.
func (kb *KeyringBuilder) Apply(keyring *credentials.GeneralOciKeyring) error {
	// set defaults
	if kb.fs == nil {
		kb.fs = osfs.New()
	}
	if len(kb.minPrivileges) == 0 {
		kb.minPrivileges = ReadOnly
	}

	// read from local path
	if len(kb.path) != 0 {
		file, err := kb.fs.Open(kb.path)
		if err != nil {
			return fmt.Errorf("unable to load config from %q: %w", kb.path, err)
		}
		defer file.Close()
		config := &SecretServerConfig{}
		if err := json.NewDecoder(file).Decode(config); err != nil {
			return fmt.Errorf("unable to decode config: %w", err)
		}
		return newKeyring(keyring, config, kb.minPrivileges, kb.forRef)
	}

	srv, err := NewSecretServer()
	if err != nil {
		if errors.Is(err, NoSecretFoundError) {
			kb.log.V(3).Info(err.Error())
			return nil
		}
		kb.log.Error(err, "unable to init secret server")
		return nil
	}
	config, err := srv.Get()
	if err != nil {
		return err
	}
	return newKeyring(keyring, config, kb.minPrivileges, kb.forRef)
}

type SecretServer struct {
	endpoint   string
	configName string

	cipherAlgorithm string
	key             []byte
}

var NoSecretFoundError = errors.New("no secret server configuration found")

const unencyptedEndpoint = "concourse-secrets/concourse_cfg"

// NewSecretServer creates a new secret server instance using given env vars.
func NewSecretServer() (*SecretServer, error) {
	keyBase64, cipher := os.Getenv(SecretKeyEnvVarName), os.Getenv(CipherEnvVarName)
	secSrvEndpoint, ccConfig := os.Getenv(EndpointEnvVarName), os.Getenv(ConcourseConfigEnvVarName)

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64 encoded key %q: %w", keyBase64, err)
	}

	if len(keyBase64) == 0 {
		ccConfig = unencyptedEndpoint
	}

	if len(secSrvEndpoint) == 0 {
		return nil, NoSecretFoundError
	}

	return &SecretServer{
		endpoint:        secSrvEndpoint,
		configName:      ccConfig,
		cipherAlgorithm: cipher,
		key:             key,
	}, nil
}

// Get returns the secret configuration from the server.
func (ss *SecretServer) Get() (*SecretServerConfig, error) {
	reader, err := ss.read()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	config := &SecretServerConfig{}
	if err := json.NewDecoder(reader).Decode(config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	return config, err
}

func (ss *SecretServer) read() (io.ReadCloser, error) {
	reader, err := getConfigFromSecretServer(ss.endpoint, ss.configName)
	if err != nil {
		return nil, err
	}

	if ss.configName == unencyptedEndpoint {
		return reader, err
	}

	switch ss.cipherAlgorithm {
	case "AES.ECB":
		var srcBuf bytes.Buffer
		if _, err := io.Copy(&srcBuf, reader); err != nil {
			return nil, fmt.Errorf("unable to read body from secret server: %w", err)
		}
		if err := reader.Close(); err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(ss.key)
		if err != nil {
			return nil, fmt.Errorf("unable to create cipher for %q; %w", ss.cipherAlgorithm, err)
		}
		dst := make([]byte, srcBuf.Len())
		if err := ECBDecrypt(block, dst, srcBuf.Bytes()); err != nil {
			return nil, err
		}
		//_ = os.WriteFile("/tmp/cc.json", dst, os.ModePerm)
		//fmt.Println(string(dst))
		//var d map[string]json.RawMessage
		//if err := json.Unmarshal(dst, &d); err != nil {
		//	panic(err)
		//}
		//fmt.Println(string(d["container_registry"]))

		return ioutil.NopCloser(bytes.NewBuffer(dst)), nil
	default:
		return nil, fmt.Errorf("unknown block %q", ss.cipherAlgorithm)
	}
}

func getConfigFromSecretServer(endpoint, configName string) (io.ReadCloser, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse secret server url %q: %w", endpoint, err)
	}
	u.Path = filepath.Join(u.Path, configName)

	res, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to get config from secret server %q: %w", u.String(), err)
	}
	return res.Body, nil
}

// newKeyring creates a new oci keyring from a config given by the reader
// if ref is defined only the credentials that match the ref are put into the keyring.
func newKeyring(keyring *credentials.GeneralOciKeyring, config *SecretServerConfig, minPriv Privilege, ref string) error {
	for key, cred := range config.ContainerRegistry {
		if minPriv == ReadWrite {
			// if no privileges are set we assume that they default to readonly.
			if cred.Privileges == ReadOnly || len(cred.Privileges) == 0 {
				continue
			}
		}

		if len(cred.Host) != 0 {
			host, err := url.Parse(cred.Host)
			if err != nil {
				return fmt.Errorf("unable to parse url %q in config %q: %w", cred.Host, key, err)
			}
			err = keyring.AddAuthConfig(host.Host, credentials.FromAuthConfig(dockerconfigtypes.AuthConfig{
				Username: cred.Username,
				Password: cred.Password,
			}, "cc-config-name", key))
			if err != nil {
				return fmt.Errorf("unable to add auth config: %w", err)
			}
		}
		for _, prefix := range cred.ImageReferencePrefixes {
			err := keyring.AddAuthConfig(prefix, credentials.FromAuthConfig(dockerconfigtypes.AuthConfig{
				Username: cred.Username,
				Password: cred.Password,
			}, "cc-config-name", key))
			if err != nil {
				return fmt.Errorf("unable to add auth config: %w", err)
			}
		}
	}

	return nil
}

// ECBDecrypt decrypts ecb data.
func ECBDecrypt(block cipher.Block, dst, src []byte) error {
	blockSize := block.BlockSize()
	if len(src)%blockSize != 0 {
		return fmt.Errorf("crypto/cipher: input not full blocks (blocksize: %d; src: %d)", blockSize, len(src))
	}
	if len(dst) < len(src) {
		return errors.New("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:blockSize])
		src = src[blockSize:]
		dst = dst[blockSize:]
	}
	return nil
}
