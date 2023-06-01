// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardenerconfig

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	gardenercfgcpi "github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/gardenerconfig/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/errors"
)

type Cipher string

const (
	Plaintext Cipher = "PLAINTEXT"
	AESECB    Cipher = "AES.ECB"
)

type Repository struct {
	ctx                       cpi.Context
	lock                      sync.RWMutex
	url                       string
	configType                gardenercfgcpi.ConfigType
	cipher                    Cipher
	key                       []byte
	propagateConsumerIdentity bool
	creds                     map[string]cpi.Credentials
	fs                        vfs.FileSystem
}

func NewRepository(ctx cpi.Context, url string, configType gardenercfgcpi.ConfigType, cipher Cipher, key []byte, propagateConsumerIdentity bool) (*Repository, error) {
	r := &Repository{
		ctx:                       ctx,
		url:                       url,
		configType:                configType,
		cipher:                    cipher,
		key:                       key,
		propagateConsumerIdentity: propagateConsumerIdentity,
		fs:                        vfsattr.Get(ctx),
	}
	if err := r.read(true); err != nil {
		return nil, fmt.Errorf("unable to read repository content: %w", err)
	}
	return r, nil
}

var _ cpi.Repository = &Repository{}

func (r *Repository) ExistsCredentials(name string) (bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.read(false); err != nil {
		return false, fmt.Errorf("unable to read repository content: %w", err)
	}

	return r.creds[name] != nil, nil
}

func (r *Repository) LookupCredentials(name string) (cpi.Credentials, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.read(false); err != nil {
		return nil, fmt.Errorf("unable to read repository content: %w", err)
	}

	auth, ok := r.creds[name]
	if !ok {
		return nil, cpi.ErrUnknownCredentials(name)
	}

	return auth, nil
}

func (r *Repository) WriteCredentials(name string, creds cpi.Credentials) (cpi.Credentials, error) {
	return nil, errors.ErrNotSupported("write", "credentials", Type)
}

func (r *Repository) read(force bool) error {
	if !force && r.creds != nil {
		return nil
	}

	configReader, err := r.getRawConfig()
	if err != nil {
		return fmt.Errorf("unable to get config: %w", err)
	}
	if configReader == nil {
		return nil
	}
	defer configReader.Close()

	handler := gardenercfgcpi.GetHandler(r.configType)
	if handler == nil {
		return errors.Newf("unable to get handler for config type %s", string(r.configType))
	}

	creds, err := handler.ParseConfig(configReader)
	if err != nil {
		return fmt.Errorf("unable to parse config: %w", err)
	}

	r.creds = map[string]cpi.Credentials{}
	for _, cred := range creds {
		credName := cred.Name()
		if _, ok := r.creds[credName]; !ok {
			r.creds[credName] = cred.Properties()
		}
		if r.propagateConsumerIdentity {
			getCredentials := func() (cpi.Credentials, error) {
				return r.LookupCredentials(credName)
			}
			cg := credentialGetter{
				getCredentials: getCredentials,
			}
			r.ctx.SetCredentialsForConsumer(cred.ConsumerIdentity(), cg)
		}
	}

	return nil
}

func (r *Repository) getRawConfig() (io.ReadCloser, error) {
	u, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url: %w", err)
	}

	var reader io.ReadCloser
	if u.Scheme == "file" {
		f, err := r.fs.Open(u.Path)
		if err != nil {
			return nil, fmt.Errorf("unable to open file: %w", err)
		}
		reader = f
	} else {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("request to secret server failed: %w", err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			// the secret server might be temporarily not available.
			// for these situations we should allow a retry at a later point in time
			// while keeping the old data for the moment.
			if errors.IsRetryable(err) {
				// TODO: log error
				return nil, nil
			}
			return nil, fmt.Errorf("request to secret server failed: %w", err)
		}
		reader = res.Body
	}

	switch r.cipher {
	case AESECB:
		var srcBuf bytes.Buffer
		if _, err := io.Copy(&srcBuf, reader); err != nil {
			return nil, fmt.Errorf("unable to read: %w", err)
		}
		if err := reader.Close(); err != nil {
			return nil, fmt.Errorf("unable to close reader: %w", err)
		}
		block, err := aes.NewCipher(r.key)
		if err != nil {
			return nil, fmt.Errorf("unable to create cipher: %w", err)
		}
		dst := make([]byte, srcBuf.Len())
		if err := ecbDecrypt(block, dst, srcBuf.Bytes()); err != nil {
			return nil, fmt.Errorf("unable to decrypt: %w", err)
		}

		return io.NopCloser(bytes.NewBuffer(dst)), nil
	case Plaintext:
		return reader, nil
	default:
		return nil, errors.ErrNotImplemented("cipher algorithm", string(r.cipher), Type)
	}
}

func ecbDecrypt(block cipher.Block, dst, src []byte) error {
	blockSize := block.BlockSize()
	if len(src)%blockSize != 0 {
		return fmt.Errorf("input must contain only full blocks (blocksize: %d; input length: %d)", blockSize, len(src))
	}
	if len(dst) < len(src) {
		return errors.New("destination is smaller than source")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:blockSize])
		src = src[blockSize:]
		dst = dst[blockSize:]
	}
	return nil
}
