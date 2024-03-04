package protocol

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"

	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

type pssParameters struct {
	Hash         pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:0"`
	MGF          pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:1"`
	SaltLength   int                      `asn1:"optional,explicit,tag:2"`
	TrailerField int                      `asn1:"optional,explicit,tag:3"` //,default:1"`
}

var oidMGF1 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 8}

func newpssParameters(hash ...crypto.Hash) (param pssParameters, err error) {

	// SHA1 is default value
	if len(hash) == 1 && hash[0] == crypto.SHA1 {
		return
	}

	var h asn1.ObjectIdentifier

	if len(hash) == 0 {
		h = oid.DigestAlgorithmSHA1
		param.SaltLength = 20
		param.TrailerField = 1
	} else {
		var ok bool
		h, ok = oid.HashToDigestAlgorithm[hash[0]]
		if !ok {
			err = errors.New("Unsupported hashfunction")
		}
	}

	hRV, err := RawValue(pkix.AlgorithmIdentifier{Algorithm: h})
	if err != nil {
		return
	}

	param.Hash = pkix.AlgorithmIdentifier{Algorithm: h}

	param.MGF = pkix.AlgorithmIdentifier{Algorithm: oidMGF1, Parameters: hRV}

	return
}

// This is needed because CheckSignature uses PSSSaltLengthEqualsHash, if PSSSaltLengthAuto is used this code is not needed
func verfiyRSAPSS(cert x509.Certificate, signatureAlgorithm pkix.AlgorithmIdentifier, signedMessage, signature []byte) (err error) {

	param, err := newpssParameters()
	if err != nil {
		return
	}

	_, err = asn1.Unmarshal(signatureAlgorithm.Parameters.FullBytes, &param)
	if err != nil {
		return
	}

	if !param.MGF.Algorithm.Equal(oidMGF1) {
		err = errors.New("Mask generator funktion not supported; only MGF1 is supported")
		return
	}

	hash := oid.DigestAlgorithmToHash[param.Hash.Algorithm.String()]

	pssOpts := rsa.PSSOptions{SaltLength: param.SaltLength, Hash: hash}

	h := hash.New()
	h.Write(signedMessage)
	digest := h.Sum(nil)

	err = rsa.VerifyPSS(cert.PublicKey.(*rsa.PublicKey), hash, digest, signature, &pssOpts)

	return
}

func newPSS(hash crypto.Hash, pub *rsa.PublicKey) (signatureAlgorithm pkix.AlgorithmIdentifier, opts *rsa.PSSOptions, err error) {

	opts = &rsa.PSSOptions{Hash: hash}

	pssParam, err := newpssParameters(hash)
	if err != nil {
		return
	}

	pssParam.SaltLength = (pub.N.BitLen()+7)/8 - 2 - hash.Size() // https://golang.org/src/crypto/rsa/pss.go?s=6982:7095#L239

	paramRV, err := RawValue(pssParam)
	if err != nil {
		return
	}
	signatureAlgorithm = pkix.AlgorithmIdentifier{Algorithm: oid.SignatureAlgorithmRSASSAPSS, Parameters: paramRV}
	return
}

// RSAESOAEPparams  ::=  SEQUENCE  {
//	hashFunc    [0] AlgorithmIdentifier DEFAULT sha1Identifier,
//	maskGenFunc [1] AlgorithmIdentifier DEFAULT mgf1SHA1Identifier,
//	pSourceFunc [2] AlgorithmIdentifier DEFAULT
//						pSpecifiedEmptyIdentifier  }
type RSAESOAEPparams struct {
	HashFunc    pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:0"`
	MaskGenFunc pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:1"`
	PSourceFunc pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:2"`
}

var oidpSpecified = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 9}

func newRSAESOAEPparams(hash ...crypto.Hash) (param RSAESOAEPparams, err error) {

	// SHA1 is default value
	if len(hash) == 1 && hash[0] == crypto.SHA1 {
		return
	}

	nullOctetString, err := RawValue([]byte{})

	var h asn1.ObjectIdentifier

	if len(hash) == 0 {
		h = oid.DigestAlgorithmSHA1
	} else {
		var ok bool
		h, ok = oid.HashToDigestAlgorithm[hash[0]]
		if !ok {
			err = errors.New("Unsupported hashfunction")
		}
	}

	hRV, err := RawValue(pkix.AlgorithmIdentifier{Algorithm: h})

	param = RSAESOAEPparams{pkix.AlgorithmIdentifier{Algorithm: h, Parameters: asn1.NullRawValue},
		pkix.AlgorithmIdentifier{Algorithm: oidMGF1, Parameters: hRV},
		pkix.AlgorithmIdentifier{Algorithm: oidpSpecified, Parameters: nullOctetString}}

	return
}

func parseRSAESOAEPparams(param []byte) (opts *rsa.OAEPOptions, err error) {
	var oaepOpts RSAESOAEPparams
	oaepOpts, err = newRSAESOAEPparams()
	if err != nil {
		return
	}

	_, err = asn1.Unmarshal(param, &oaepOpts)
	if err != nil {
		return
	}

	opts = &rsa.OAEPOptions{Hash: oid.DigestAlgorithmToHash[oaepOpts.HashFunc.Algorithm.String()], Label: []byte{}}

	if !oaepOpts.MaskGenFunc.Algorithm.Equal(oidMGF1) {
		err = errors.New("Unsupported mask generation funktion" + oaepOpts.MaskGenFunc.Algorithm.String())
		return
	}

	if !oaepOpts.PSourceFunc.Algorithm.Equal(oidpSpecified) {
		err = errors.New("Unsupported p source funktion" + oaepOpts.PSourceFunc.Algorithm.String())
		return
	}

	return
}

func isRSAPSS(cert *x509.Certificate) bool {
	switch cert.SignatureAlgorithm {
	case x509.SHA256WithRSAPSS, x509.SHA384WithRSAPSS, x509.SHA512WithRSAPSS:
		return true
	default:
		return false
	}
}
