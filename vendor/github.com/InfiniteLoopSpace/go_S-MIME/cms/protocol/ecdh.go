package protocol

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

var errUnsupported = errors.New("Unsupported hash function")

// ECDHsharedSecret computes shared secret with ephemeral static ECDH
func ECDHsharedSecret(curve elliptic.Curve, priv []byte, pubX, pubY *big.Int) []byte {

	x, _ := curve.ScalarMult(pubX, pubY, priv)

	return x.Bytes()
}

// ANSIx963KDF implents ANSI X9.63 key derivation function
func ANSIx963KDF(sharedSecret, sharedInfo []byte, keyLen int, hash crypto.Hash) (key []byte, err error) {

	ctr := make([]byte, 4)
	ctr[3] = 0x01
	if hash == 0 || !hash.Available() {
		return nil, errUnsupported
	}
	h := hash.New()

	for i := 0; i < keyLen/hash.Size()+1; i++ {
		h.Reset()
		h.Write(sharedSecret)
		h.Write(ctr)
		h.Write(sharedInfo)
		key = append(key, h.Sum(nil)...)

		// Increment counter
		for i := len(ctr) - 1; i >= 0; i-- {
			ctr[i]++
			if ctr[i] != 0 {
				break
			}
		}
	}

	return key[:keyLen], nil
}

func encryptKeyECDH(key []byte, recipient *x509.Certificate) (kari KeyAgreeRecipientInfo, err error) {

	keyWrapAlgorithm := oid.KeyWrap{KeyWrapAlgorithm: oid.AES128Wrap}
	keyEncryptionAlgorithm := oid.DHSinglePassstdDHsha256kdfscheme
	hash := oid.KDFHashAlgorithm[keyEncryptionAlgorithm.String()]

	kari.UKM = make([]byte, 8)
	rand.Read(kari.UKM)

	kari.Version = 3
	kari.Originator.OriginatorKey.Algorithm = pkix.AlgorithmIdentifier{Algorithm: oid.ECPublicKey}

	// check recipient key

	if recipient.PublicKeyAlgorithm != x509.ECDSA {
		err = errors.New("Recipient certficiate has wrong public key algorithm, expected ECDSA")
		return
	}

	pubKey, ok := recipient.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		err = errors.New("Can not parse public key of recipient")
		return
	}

	// genrate ephemeral public key and key encryption key

	priv, x, y, err := elliptic.GenerateKey(pubKey.Curve, rand.Reader)
	if err != nil {
		return
	}

	ephPubKey := elliptic.Marshal(pubKey.Curve, x, y)
	kari.Originator.OriginatorKey.PublicKey = asn1.BitString{Bytes: ephPubKey, BitLength: len(ephPubKey) * 8}

	sharedSecret := ECDHsharedSecret(pubKey.Curve, priv, pubKey.X, pubKey.Y)

	keyLenBigEnd := make([]byte, 4)
	binary.BigEndian.PutUint32(keyLenBigEnd, uint32(keyWrapAlgorithm.KeyLen())*8)
	sharedInfo := ECCCMSSharedInfo{KeyInfo: keyWrapAlgorithm.AlgorithmIdentifier(),
		SuppPubInfo: keyLenBigEnd,
		EntityUInfo: kari.UKM}

	sharedInfoDER, err := asn1.Marshal(sharedInfo)

	kek, err := ANSIx963KDF(sharedSecret, sharedInfoDER, keyWrapAlgorithm.KeyLen(), hash)
	if err != nil {
		return
	}

	// encrypt key

	keyWrapAlgorithm.KEK = kek
	encKey, err := keyWrapAlgorithm.Wrap(key)
	if err != nil {
		return
	}

	keyWrapAlgorithmIdentifier, err := RawValue(keyWrapAlgorithm.AlgorithmIdentifier())
	if err != nil {
		return
	}

	kari.KeyEncryptionAlgorithm = pkix.AlgorithmIdentifier{Algorithm: keyEncryptionAlgorithm,
		Parameters: keyWrapAlgorithmIdentifier}

	ias, err := NewIssuerAndSerialNumber(recipient)
	karID := KeyAgreeRecipientIdentifier{IAS: ias}

	kari.RecipientEncryptedKeys = append(kari.RecipientEncryptedKeys, RecipientEncryptedKey{RID: karID, EncryptedKey: encKey})

	return
}

// ECCCMSSharedInfo ECC-CMS-SharedInfo ::= SEQUENCE {
//	keyInfo         AlgorithmIdentifier,
//	entityUInfo [0] EXPLICIT OCTET STRING OPTIONAL,
//	suppPubInfo [2] EXPLICIT OCTET STRING  }
type ECCCMSSharedInfo struct {
	KeyInfo     pkix.AlgorithmIdentifier
	EntityUInfo []byte `asn1:"optional,explicit,tag:0"`
	SuppPubInfo []byte `asn1:"explicit,tag:2"`
}

func (kari *KeyAgreeRecipientInfo) decryptKey(keyPair tls.Certificate) (key []byte, err error) {

	// check for ec key

	if kari.Version != 3 {
		err = errors.New("Version not supported")
		return
	}

	if !kari.Originator.OriginatorKey.Algorithm.Algorithm.Equal(oid.ECPublicKey) {
		err = errors.New("Orginator key algorithm not supported")
		return
	}

	pubKey, ok := keyPair.Leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		err = errors.New("Can not parse public key of recipient")
		return
	}

	x, y := elliptic.Unmarshal(pubKey.Curve, kari.Originator.OriginatorKey.PublicKey.Bytes)

	// genrate ephemeral public key and key encryption key

	priv := keyPair.PrivateKey.(*ecdsa.PrivateKey)

	privateKeyBytes := keyPair.PrivateKey.(*ecdsa.PrivateKey).D.Bytes()
	paddedPrivateKey := make([]byte, (priv.Curve.Params().N.BitLen()+7)/8)
	copy(paddedPrivateKey[len(paddedPrivateKey)-len(privateKeyBytes):], privateKeyBytes)

	sharedSecret := ECDHsharedSecret(pubKey.Curve, paddedPrivateKey, x, y)

	hash, ok := oid.KDFHashAlgorithm[kari.KeyEncryptionAlgorithm.Algorithm.String()]
	if !ok {
		err = errors.New("Unsupported key derivation hash algorithm")
		return
	}

	var keyWrapAlgorithmIdentifier pkix.AlgorithmIdentifier
	asn1.Unmarshal(kari.KeyEncryptionAlgorithm.Parameters.FullBytes, &keyWrapAlgorithmIdentifier)
	keyWrapAlgorithm := oid.KeyWrap{KeyWrapAlgorithm: keyWrapAlgorithmIdentifier.Algorithm}

	//

	keyLenBigEnd := make([]byte, 4)
	binary.BigEndian.PutUint32(keyLenBigEnd, uint32(keyWrapAlgorithm.KeyLen())*8)
	sharedInfo := ECCCMSSharedInfo{KeyInfo: keyWrapAlgorithmIdentifier,
		SuppPubInfo: keyLenBigEnd,
		EntityUInfo: kari.UKM}

	sharedInfoDER, err := asn1.Marshal(sharedInfo)

	kek, err := ANSIx963KDF(sharedSecret, sharedInfoDER, keyWrapAlgorithm.KeyLen(), hash)
	if err != nil {
		return
	}

	keyWrapAlgorithm.KEK = kek

	// encrypt key

	ias, err := NewIssuerAndSerialNumber(keyPair.Leaf)
	if err != nil {
		return
	}

	for i := range kari.RecipientEncryptedKeys {
		if kari.RecipientEncryptedKeys[i].RID.IAS.Equal(ias) {
			key, err = keyWrapAlgorithm.UnWrap(kari.RecipientEncryptedKeys[i].EncryptedKey)
			return
		}
	}

	err = ErrNoKeyFound

	return
}
