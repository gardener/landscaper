// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"github.com/open-component-model/ocm/pkg/encrypt"
	"github.com/open-component-model/ocm/pkg/errors"
)

const DECRYPTION_PREFIX = "decrypt:"

const KIND_DECRYPTION_KEY = "decryption key"

func DecryptionKeyName(name string) string {
	return DECRYPTION_PREFIX + name
}

func ResolvePrivateKey(reg KeyRegistryFuncs, name string) (interface{}, error) {
	key := reg.GetPrivateKey(name)
	if key == nil {
		return nil, nil
	}

	data, ok := key.([]byte)
	if !ok {
		if str, ok := key.(string); ok {
			data = []byte(str)
		}
	}
	if data == nil {
		return key, nil
	}

	data, algo := encrypt.GetEncyptedData(data)
	if data == nil {
		return key, nil
	}

	encryptionKey, err := ResolvePrivateKey(reg, DecryptionKeyName(name))
	if err != nil {
		return nil, err
	}

	if encryptionKey == nil {
		return nil, errors.ErrNotFound(KIND_DECRYPTION_KEY, DecryptionKeyName(name))
	}
	var keyData []byte
	if raw, ok := encryptionKey.([]byte); ok {
		keyData, err = encrypt.KeyFromPem(raw)
		if err != nil {
			keyData = raw
		}
	} else {
		return nil, errors.ErrInvalid(KIND_DECRYPTION_KEY, DecryptionKeyName(name))
	}
	if err := algo.CheckKey(keyData); err != nil {
		return nil, err
	}
	return encrypt.Decrypt(keyData, data)
}
