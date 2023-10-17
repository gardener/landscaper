// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/pem"
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	PEM_ENCRYPTION_KEY = "ENCRYPTION KEY"
	PEM_ENCRYPTED_DATA = "ENCRYPTED DATA"
)

type algorithm byte

const (
	AES_128 = algorithm(16)
	AES_192 = algorithm(24)
	AES_256 = algorithm(32)
)

const ALGO = "algorithm"

const (
	ALGO_AES_128 = "AES-128"
	ALGO_AES_192 = "AES-192"
	ALGO_AES_256 = "AES-256"
)

var algos = map[algorithm]string{
	AES_128: ALGO_AES_128,
	AES_192: ALGO_AES_192,
	AES_256: ALGO_AES_256,
}

var name2algo = map[string]algorithm{
	ALGO_AES_128: AES_128,
	ALGO_AES_192: AES_192,
	ALGO_AES_256: AES_256,
}

func (a algorithm) String() string {
	return algos[a]
}

func (a algorithm) KeyLength() int {
	return int(a)
}

func (a algorithm) CheckKey(key []byte) error {
	if a.KeyLength() != len(key) {
		return aes.KeySizeError(len(key))
	}
	return nil
}

func AlgoForKey(key []byte) (algorithm, error) {
	for a := range algos {
		if len(key) == a.KeyLength() {
			return a, nil
		}
	}
	return 0, aes.KeySizeError(len(key))
}

func NewKey(t algorithm) ([]byte, error) {
	key := make([]byte, t)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func Encrypt(key []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func Decrypt(key []byte, cipherText []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := cipherText[:gcm.NonceSize()]
	cipherText = cipherText[gcm.NonceSize():]
	return gcm.Open(nil, nonce, cipherText, nil)
}

func KeyFromPem(data []byte) ([]byte, error) {
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Type == PEM_ENCRYPTION_KEY {
			return block.Bytes, nil
		}
	}
	return nil, errors.ErrNotFound("pem block", PEM_ENCRYPTION_KEY)
}

func KeyToPem(data []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:    PEM_ENCRYPTION_KEY,
		Headers: nil,
		Bytes:   data,
	})
}

func KeyFromAny(k interface{}) ([]byte, error) {
	if data, ok := k.([]byte); ok {
		for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
			if block.Type == PEM_ENCRYPTION_KEY {
				return block.Bytes, nil
			}
		}
		return data, nil
	}
	return nil, errors.ErrUnknown(PEM_ENCRYPTION_KEY)
}

func GetEncyptedData(data []byte) ([]byte, algorithm) {
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Type == PEM_ENCRYPTED_DATA {
			algo := AES_256
			if block.Headers != nil {
				if name := block.Headers[ALGO]; name != "" {
					algo = name2algo[name]
				}
			}
			return block.Bytes, algo
		}
	}
	return nil, 0
}

func EncryptedToPem(algo algorithm, data []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:    PEM_ENCRYPTED_DATA,
		Headers: map[string]string{ALGO: algo.String()},
		Bytes:   data,
	})
}

func OptionalDecrypt(key []byte, data []byte) ([]byte, error) {
	cipherText, algo := GetEncyptedData(data)
	if cipherText != nil {
		if len(key) != algo.KeyLength() {
			return nil, aes.KeySizeError(len(key))
		}
		return Decrypt(key, cipherText)
	}
	return data, nil
}

func WriteKey(key []byte, path string, fss ...vfs.FileSystem) error {
	data := KeyToPem(key)
	return vfs.WriteFile(accessio.FileSystem(fss...), path, data, 0o100)
}

func ReadKey(path string, fss ...vfs.FileSystem) ([]byte, error) {
	data, err := vfs.ReadFile(accessio.FileSystem(fss...), path)
	if err != nil {
		return nil, err
	}
	return KeyFromPem(data)
}
