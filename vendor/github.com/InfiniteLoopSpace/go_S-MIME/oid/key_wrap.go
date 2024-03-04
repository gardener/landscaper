package oid

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"errors"
)

// KeyWrap wraps and unwraps key with the key encrytion key (KEK) for a given (KeyWrapAlgorithm)
type KeyWrap struct {
	KEK              []byte
	KeyWrapAlgorithm asn1.ObjectIdentifier
}

// Wrap wraps the content encryption key (cek)
func (kw *KeyWrap) Wrap(cek []byte) (ciphertext []byte, err error) {

	var block cipher.Block
	switch kw.KeyWrapAlgorithm.String() {
	case AES128Wrap.String(), AES192Wrap.String(), AES256Wrap.String():
		block, err = aes.NewCipher(kw.KEK)
		if err != nil {
			return
		}
	}

	return Wrap(block, cek)

}

// UnWrap unwraps the encrypted key (encKey)
func (kw *KeyWrap) UnWrap(encKey []byte) (cek []byte, err error) {

	var block cipher.Block
	switch kw.KeyWrapAlgorithm.String() {
	case AES128Wrap.String(), AES192Wrap.String(), AES256Wrap.String():
		block, err = aes.NewCipher(kw.KEK)
		if err != nil {
			return
		}
	}

	return Unwrap(block, encKey)

}

// KeyLen returns the key lenght of the key wrap algorithm
func (kw *KeyWrap) KeyLen() (len int) {

	switch kw.KeyWrapAlgorithm.String() {
	case AES128Wrap.String():
		len = 16
	case AES192Wrap.String():
		len = 24
	case AES256Wrap.String():
		len = 32
	}

	return
}

// AlgorithmIdentifier returns the OID of the key wrap algorithm
func (kw *KeyWrap) AlgorithmIdentifier() (algID pkix.AlgorithmIdentifier) {

	switch kw.KeyWrapAlgorithm.String() {
	case AES128Wrap.String(), AES192Wrap.String(), AES256Wrap.String():
		algID = pkix.AlgorithmIdentifier{Algorithm: kw.KeyWrapAlgorithm}
	}

	return
}

// defaultIV from RFC-3394
var defaultIV = []byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}

// Wrap encrypts the content encryption key (cek) with the given AES cipher (block), using the AES Key Wrap algorithm (RFC-3394)
func Wrap(block cipher.Block, cek []byte) (encKey []byte, err error) {
	if len(cek)%8 != 0 {
		return nil, errors.New("Lenght of cek must be in 8-byte blocks")
	}

	// 1. Initialize variables

	// Set A = IV, an initial value (see 2.2.3)
	B := make([]byte, 16)
	copy(B, defaultIV)

	// For i = 1 to n
	// R[i] = P[i]
	encKey = make([]byte, len(cek)+8)
	copy(encKey[8:], cek)

	n := len(cek) / 8

	// 2. Calculate intermediate values.
	for j := 0; j <= 5; j++ {
		for i := 1; i <= n; i++ {

			// B = AES(K, A | R[i])
			copy(B[8:], encKey[i*8:(i+1)*8])
			block.Encrypt(B, B)

			// A = MSB(64, B) ^ t where t = (n*j)+i
			t := uint64(n*j + i)
			b := binary.BigEndian.Uint64(B[:8]) ^ t
			binary.BigEndian.PutUint64(B[:8], b)

			// R[i] = LSB(64, B)
			copy(encKey[i*8:(i+1)*8], B[8:])
		}
	}

	// 3. Output the results.
	copy(encKey[:8], B[:8])
	return
}

// Unwrap decrypts the provided encrypted key (encKey) with the given AES cipher (block), using the AES Key Wrap algorithm (RFC-3394).
// Returns an error if validation fails.
func Unwrap(block cipher.Block, encKey []byte) (cek []byte, err error) {
	if len(cek)%8 != 0 {
		return nil, errors.New("Length of encKey must multiple 8-bytes")
	}

	//Initialize variables
	B := make([]byte, 16)
	copy(B, encKey[:8])

	cek = make([]byte, len(encKey)-8)
	copy(cek, encKey[8:])

	n := (len(encKey) / 8) - 1

	//Compute intermediate values
	for j := 5; j >= 0; j-- {
		for i := n; i >= 1; i-- {

			// B = AES-1(K, (A ^ t) | R[i]) where t = n*j+i
			copy(B[8:], cek[(i-1)*8:i*8])
			t := uint64(n*j + i)
			b := binary.BigEndian.Uint64(B[:8]) ^ t
			binary.BigEndian.PutUint64(B[:8], b)

			block.Decrypt(B, B)

			// A = MSB(64, B)
			// R[i] = LSB(64, B)
			copy(cek[(i-1)*8:i*8], B[8:])

		}
	}

	if subtle.ConstantTimeCompare(B[:8], defaultIV) != 1 {
		return nil, errors.New("Integrity check failed - unexpected IV")
	}

	//Output
	return
}
