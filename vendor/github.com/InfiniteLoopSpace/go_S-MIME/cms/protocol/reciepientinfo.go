package protocol

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"log"
	"time"

	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

//RecipientInfo ::= CHOICE {
//	ktri KeyTransRecipientInfo,
//	kari [1] KeyAgreeRecipientInfo,
//	kekri [2] KEKRecipientInfo,
//	pwri [3] PasswordRecipientInfo,
//	ori [4] OtherRecipientInfo }
type RecipientInfo struct {
	KTRI  KeyTransRecipientInfo `asn1:"optional"`
	KARI  KeyAgreeRecipientInfo `asn1:"optional,tag:1"` //KeyAgreeRecipientInfo
	KEKRI asn1.RawValue         `asn1:"optional,tag:2"`
	PWRI  asn1.RawValue         `asn1:"optional,tag:3"`
	ORI   asn1.RawValue         `asn1:"optional,tag:4"`
}

func (recInfo *RecipientInfo) decryptKey(keyPair tls.Certificate) (key []byte, err error) {

	key, err = recInfo.KTRI.decryptKey(keyPair)
	if key != nil || (err != nil && err != ErrUnsupported) {
		return
	}

	key, err = recInfo.KARI.decryptKey(keyPair)

	return
}

//KeyTransRecipientInfo ::= SEQUENCE {
//	version CMSVersion,  -- always set to 0 or 2
//	rid RecipientIdentifier,
//	keyEncryptionAlgorithm KeyEncryptionAlgorithmIdentifier,
//	encryptedKey EncryptedKey }
type KeyTransRecipientInfo struct {
	Version                int
	Rid                    RecipientIdentifier `asn1:"choice"`
	KeyEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedKey           []byte
}

func (ktri *KeyTransRecipientInfo) decryptKey(keyPair tls.Certificate) (key []byte, err error) {

	ias, err := NewIssuerAndSerialNumber(keyPair.Leaf)
	if err != nil {
		return
	}

	ski := keyPair.Leaf.SubjectKeyId

	certPubAlg := oid.PublicKeyAlgorithmToEncrytionAlgorithm[keyPair.Leaf.PublicKeyAlgorithm].Algorithm

	var decOpts crypto.DecrypterOpts
	pkcs15CertwithOAEP := false

	if ktri.KeyEncryptionAlgorithm.Algorithm.Equal(oid.EncryptionAlgorithmRSAESOAEP) {

		if certPubAlg.Equal(oid.EncryptionAlgorithmRSA) {
			pkcs15CertwithOAEP = true
		}

		decOpts, err = parseRSAESOAEPparams(ktri.KeyEncryptionAlgorithm.Parameters.FullBytes)
		if err != nil {
			return
		}
	}

	//version is the syntax version number.  If the SignerIdentifier is
	//the CHOICE issuerAndSerialNumber, then the version MUST be 1.  If
	//the SignerIdentifier is subjectKeyIdentifier, then the version
	//MUST be 3.
	switch ktri.Version {
	case 0:
		if ias.Equal(ktri.Rid.IAS) {
			if ktri.KeyEncryptionAlgorithm.Algorithm.Equal(certPubAlg) || pkcs15CertwithOAEP {

				decrypter := keyPair.PrivateKey.(crypto.Decrypter)
				return decrypter.Decrypt(rand.Reader, ktri.EncryptedKey, decOpts)

			}
			log.Println("Key encrytion algorithm not matching")
		}
	case 2:
		if bytes.Equal(ski, ktri.Rid.SKI) {
			if ktri.KeyEncryptionAlgorithm.Algorithm.Equal(certPubAlg) || pkcs15CertwithOAEP {

				decrypter := keyPair.PrivateKey.(crypto.Decrypter)
				return decrypter.Decrypt(rand.Reader, ktri.EncryptedKey, decOpts)

			}
			log.Println("Key encrytion algorithm not matching")
		}
	default:
		return nil, ErrUnsupported
	}

	return nil, nil
}

//RecipientIdentifier ::= CHOICE {
//	issuerAndSerialNumber IssuerAndSerialNumber,
//	subjectKeyIdentifier [0] SubjectKeyIdentifier }
type RecipientIdentifier struct {
	IAS IssuerAndSerialNumber `asn1:"optional"`
	SKI []byte                `asn1:"optional,tag:0"`
}

// NewRecipientInfo creates RecipientInfo for giben recipient and key.
func NewRecipientInfo(recipient *x509.Certificate, key []byte) (info RecipientInfo, err error) {

	switch recipient.PublicKeyAlgorithm {
	case x509.RSA:
		var ktri KeyTransRecipientInfo
		ktri, err = encryptKeyRSA(key, recipient)
		if err != nil {
			return
		}
		info = RecipientInfo{KTRI: ktri}
	case x509.ECDSA:
		var kari KeyAgreeRecipientInfo
		kari, err = encryptKeyECDH(key, recipient)
		if err != nil {
			return
		}
		info = RecipientInfo{KARI: kari}
	default:
		err = errors.New("Public key algorithm not supported")
	}

	return
}

func encryptKeyRSA(key []byte, recipient *x509.Certificate) (ktri KeyTransRecipientInfo, err error) {
	ktri.Version = 0 //issuerAndSerialNumber

	switch ktri.Version {
	case 0:
		ias, err := NewIssuerAndSerialNumber(recipient)
		if err != nil {
			log.Fatal(err)
		}
		ktri.Rid.IAS = ias
	case 2:
		ktri.Rid.SKI = recipient.SubjectKeyId
	}

	if pub := recipient.PublicKey.(*rsa.PublicKey); pub != nil {

		if isRSAPSS(recipient) {
			hash := crypto.SHA256
			var oaepparam RSAESOAEPparams
			oaepparam, err = newRSAESOAEPparams(hash)
			if err != nil {
				return
			}
			var oaepparamRV asn1.RawValue
			oaepparamRV, err = RawValue(oaepparam)
			if err != nil {
				return
			}
			ktri.KeyEncryptionAlgorithm = pkix.AlgorithmIdentifier{Algorithm: oid.EncryptionAlgorithmRSAESOAEP, Parameters: oaepparamRV}
			h := hash.New()
			ktri.EncryptedKey, err = rsa.EncryptOAEP(h, rand.Reader, pub, key, nil)
			return
		}

		ktri.KeyEncryptionAlgorithm = pkix.AlgorithmIdentifier{Algorithm: oid.EncryptionAlgorithmRSA}
		ktri.EncryptedKey, err = rsa.EncryptPKCS1v15(rand.Reader, pub, key)
		return
	}

	err = ErrUnsupportedAlgorithm
	return
}

// ErrUnsupportedAlgorithm is returned if the algorithm is unsupported.
var ErrUnsupportedAlgorithm = errors.New("cms: cannot decrypt data: unsupported algorithm")

//KeyAgreeRecipientInfo ::= SEQUENCE {
//	version CMSVersion,  -- always set to 3
//	originator [0] EXPLICIT OriginatorIdentifierOrKey,
//	ukm [1] EXPLICIT UserKeyingMaterial OPTIONAL,
//	keyEncryptionAlgorithm KeyEncryptionAlgorithmIdentifier,
//	recipientEncryptedKeys RecipientEncryptedKeys }
type KeyAgreeRecipientInfo struct {
	Version                int
	Originator             OriginatorIdentifierOrKey `asn1:"explicit,choice,tag:0"`
	UKM                    []byte                    `asn1:"explicit,optional,tag:1"`
	KeyEncryptionAlgorithm pkix.AlgorithmIdentifier  ``
	RecipientEncryptedKeys []RecipientEncryptedKey   `asn1:"sequence"` //RecipientEncryptedKeys ::= SEQUENCE OF RecipientEncryptedKey
}

//OriginatorIdentifierOrKey ::= CHOICE {
//	issuerAndSerialNumber IssuerAndSerialNumber,
//	subjectKeyIdentifier [0] SubjectKeyIdentifier,
//	originatorKey [1] OriginatorPublicKey }
type OriginatorIdentifierOrKey struct {
	IAS           IssuerAndSerialNumber `asn1:"optional"`
	SKI           []byte                `asn1:"optional,tag:0"`
	OriginatorKey OriginatorPublicKey   `asn1:"optional,tag:1"`
}

//OriginatorPublicKey ::= SEQUENCE {
//	algorithm AlgorithmIdentifier,
//	publicKey BIT STRING
type OriginatorPublicKey struct {
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

//RecipientEncryptedKey ::= SEQUENCE {
//	rid KeyAgreeRecipientIdentifier,
//	encryptedKey EncryptedKey }
type RecipientEncryptedKey struct {
	RID          KeyAgreeRecipientIdentifier `asn1:"choice"`
	EncryptedKey []byte
}

//KeyAgreeRecipientIdentifier ::= CHOICE {
//	issuerAndSerialNumber IssuerAndSerialNumber,
//	rKeyId [0] IMPLICIT RecipientKeyIdentifier }
type KeyAgreeRecipientIdentifier struct {
	IAS    IssuerAndSerialNumber  `asn1:"optional"`
	RKeyID RecipientKeyIdentifier `asn1:"optional,tag:0"`
}

//RecipientKeyIdentifier ::= SEQUENCE {
//	subjectKeyIdentifier SubjectKeyIdentifier,
//	date GeneralizedTime OPTIONAL,
//	other OtherKeyAttribute OPTIONAL }
type RecipientKeyIdentifier struct {
	SubjectKeyIdentifier []byte            //SubjectKeyIdentifier ::= OCTET STRING
	Date                 time.Time         `asn1:"optional"`
	Other                OtherKeyAttribute `asn1:"optional"`
}

//OtherKeyAttribute ::= SEQUENCE {
//	keyAttrId OBJECT IDENTIFIER,
//	keyAttr ANY DEFINED BY keyAttrId OPTIONAL }
type OtherKeyAttribute struct {
	KeyAttrID asn1.ObjectIdentifier
	KeyAttr   asn1.RawValue `asn1:"optional"`
}
