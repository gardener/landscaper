package oid

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptionAlgorithm does the handling of the encrypton and decryption for a given algorithm identifier.
type EncryptionAlgorithm struct {
	EncryptionAlgorithmIdentifier        asn1.ObjectIdentifier
	ContentEncryptionAlgorithmIdentifier pkix.AlgorithmIdentifier
	Key, IV, MAC                         []byte
}

// Encryption Algorithm OIDs
var (
	EncryptionAlgorithmDESCBC     = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 7}
	EncryptionAlgorithmDESEDE3CBC = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
	EncryptionAlgorithmAES128CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	EncryptionAlgorithmAES256CBC  = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	//AEAD
	EncryptionAlgorithmAES128GCM = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 6}
	AEADChaCha20Poly1305         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 3, 18}
)

// SymmetricKeyLen maps the encryption algorithm to its key length
var SymmetricKeyLen = map[string]int{
	EncryptionAlgorithmDESCBC.String():     8,
	EncryptionAlgorithmDESEDE3CBC.String(): 24,
	EncryptionAlgorithmAES128CBC.String():  16,
	EncryptionAlgorithmAES256CBC.String():  32,
	//AEAD
	EncryptionAlgorithmAES128GCM.String(): 16,
	AEADChaCha20Poly1305.String():         32,
}

// Encrypt encrypts the plaintext and returns the ciphertext.
func (e *EncryptionAlgorithm) Encrypt(plaintext []byte) (ciphertext []byte, err error) {

	if e.Key == nil {
		e.Key = make([]byte, SymmetricKeyLen[e.EncryptionAlgorithmIdentifier.String()])
		rand.Read(e.Key)
	}

	//Choose cipher
	var blockCipher cipher.Block

	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String(), EncryptionAlgorithmAES128GCM.String():
		blockCipher, err = aes.NewCipher(e.Key)
		if err != nil {
			return
		}
	case AEADChaCha20Poly1305.String():
	default:
		err = errors.New("Content encrytion: cipher not supportet")
		return
	}

	//Choose blockmode
	var blockMode cipher.BlockMode
	var aead cipher.AEAD
	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String():
		if e.IV == nil {
			e.IV = make([]byte, blockCipher.BlockSize())
			rand.Read(e.IV)
		}

		blockMode = cipher.NewCBCEncrypter(blockCipher, e.IV)
		e.ContentEncryptionAlgorithmIdentifier = pkix.AlgorithmIdentifier{
			Algorithm:  e.EncryptionAlgorithmIdentifier,
			Parameters: asn1.RawValue{Tag: 4, Bytes: e.IV}}
	case EncryptionAlgorithmAES128GCM.String():
		aead, err = cipher.NewGCM(blockCipher)
		if err != nil {
			return
		}
	case AEADChaCha20Poly1305.String():
		aead, err = chacha20poly1305.New(e.Key)
		if err != nil {
			return
		}
	}

	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String():
		var plain []byte
		plain, err = pad(plaintext, blockCipher.BlockSize())
		if err != nil {
			return
		}

		ciphertext = make([]byte, len(plain))

		blockMode.CryptBlocks(ciphertext, plain)

		return
	case EncryptionAlgorithmAES128GCM.String(), AEADChaCha20Poly1305.String():
		nonce := make([]byte, nonceSize)
		_, err = rand.Read(nonce)
		if err != nil {
			return
		}

		ciphertext = aead.Seal(nil, nonce, plaintext, nil)

		e.MAC = ciphertext[len(ciphertext)-aead.Overhead():]
		ciphertext = ciphertext[:len(ciphertext)-aead.Overhead()]
		switch e.EncryptionAlgorithmIdentifier.String() {
		case EncryptionAlgorithmAES128GCM.String():
			paramSeq := aesGCMParameters{
				Nonce:  nonce,
				ICVLen: aead.Overhead(),
			}

			paramBytes, _ := asn1.Marshal(paramSeq)

			e.ContentEncryptionAlgorithmIdentifier = pkix.AlgorithmIdentifier{
				Algorithm: e.EncryptionAlgorithmIdentifier,
				Parameters: asn1.RawValue{
					Tag:   asn1.TagSequence,
					Bytes: paramBytes,
				}}
		case AEADChaCha20Poly1305.String():
			e.ContentEncryptionAlgorithmIdentifier = pkix.AlgorithmIdentifier{
				Algorithm:  e.EncryptionAlgorithmIdentifier,
				Parameters: asn1.RawValue{Tag: 4, Bytes: nonce}}
		}

	}

	return
}

const nonceSize = 12

type aesGCMParameters struct {
	Nonce  []byte `asn1:"tag:4"`
	ICVLen int
}

// Decrypt decrypts the ciphertext and returns the plaintext.
func (e *EncryptionAlgorithm) Decrypt(ciphertext []byte) (plaintext []byte, err error) {

	e.EncryptionAlgorithmIdentifier = e.ContentEncryptionAlgorithmIdentifier.Algorithm

	//Choose cipher
	var blockCipher cipher.Block

	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String(), EncryptionAlgorithmAES128GCM.String():
		blockCipher, err = aes.NewCipher(e.Key)
		if err != nil {
			return
		}
	case EncryptionAlgorithmDESCBC.String():
		blockCipher, err = des.NewCipher(e.Key)
		fmt.Println("Warning: message is encoded with DES. DES should NOT be used.")
	case EncryptionAlgorithmDESEDE3CBC.String():
		blockCipher, err = des.NewTripleDESCipher(e.Key)
		fmt.Println("Warning: message is encoded with 3DES. 3DES should NOT be used.")
	case AEADChaCha20Poly1305.String():
	default:
		err = errors.New("Content encrytion: cipher not supportet")
		return
	}

	//Choose blockmode
	var blockMode cipher.BlockMode
	var aead cipher.AEAD
	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String(), EncryptionAlgorithmDESCBC.String(), EncryptionAlgorithmDESEDE3CBC.String():
		e.IV = e.ContentEncryptionAlgorithmIdentifier.Parameters.Bytes
		blockMode = cipher.NewCBCDecrypter(blockCipher, e.IV)
	case EncryptionAlgorithmAES128GCM.String():
		aead, err = cipher.NewGCM(blockCipher)
		if err != nil {
			return
		}
	case AEADChaCha20Poly1305.String():
		aead, err = chacha20poly1305.New(e.Key)
		if err != nil {
			return
		}
	}

	switch e.EncryptionAlgorithmIdentifier.String() {
	case EncryptionAlgorithmAES128CBC.String(), EncryptionAlgorithmAES256CBC.String(), EncryptionAlgorithmDESCBC.String(), EncryptionAlgorithmDESEDE3CBC.String():
		plaintext = make([]byte, len(ciphertext))
		blockMode.CryptBlocks(plaintext, ciphertext)
		return unpad(plaintext, blockMode.BlockSize())
	case EncryptionAlgorithmAES128GCM.String(), AEADChaCha20Poly1305.String():
		var cipher []byte
		cipher = append(cipher, ciphertext...)
		cipher = append(cipher, e.MAC...)

		var nonce []byte
		switch e.EncryptionAlgorithmIdentifier.String() {
		case EncryptionAlgorithmAES128GCM.String():
			params := aesGCMParameters{}
			paramBytes := e.ContentEncryptionAlgorithmIdentifier.Parameters.Bytes
			_, err = asn1.Unmarshal(paramBytes, &params)
			if err != nil {
				return nil, err
			}
			nonce = params.Nonce
		case AEADChaCha20Poly1305.String():
			nonce = e.ContentEncryptionAlgorithmIdentifier.Parameters.Bytes
		}

		plaintext, err = aead.Open(nil, nonce, cipher, nil)
		return
	}
	return
}

func pad(data []byte, blocklen int) ([]byte, error) {
	if blocklen < 1 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}
	padlen := blocklen - (len(data) % blocklen)
	if padlen == 0 {
		padlen = blocklen
	}
	pad := bytes.Repeat([]byte{byte(padlen)}, padlen)
	return append(data, pad...), nil
}

func unpad(data []byte, blocklen int) ([]byte, error) {
	if blocklen < 1 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}
	if len(data)%blocklen != 0 || len(data) == 0 {
		return nil, fmt.Errorf("invalid data len %d", len(data))
	}

	// the last byte is the length of padding
	padlen := int(data[len(data)-1])
	if padlen > blocklen {
		return nil, fmt.Errorf("pad len %d is bigger than block len len %d", padlen, blocklen)
	}

	// check padding integrity, all bytes should be the same
	pad := data[len(data)-padlen:]
	for _, padbyte := range pad {
		if padbyte != byte(padlen) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padlen], nil
}
