package protocol

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

// SignedDataContent returns SignedData if ContentType is SignedData.
func (ci ContentInfo) SignedDataContent() (*SignedData, error) {
	if !ci.ContentType.Equal(oid.SignedData) {
		return nil, ErrWrongType
	}

	sd := new(SignedData)
	if rest, err := asn.Unmarshal(ci.Content.Bytes, sd); err != nil {
		return nil, err
	} else if len(rest) > 0 {
		return nil, ErrTrailingData
	}

	return sd, nil
}

// SignedData ::= SEQUENCE {
//   version CMSVersion,
//   digestAlgorithms DigestAlgorithmIdentifiers,
//   encapContentInfo EncapsulatedContentInfo,
//   certificates [0] IMPLICIT CertificateSet OPTIONAL,
//   crls [1] IMPLICIT RevocationInfoChoices OPTIONAL,
//   signerInfos SignerInfos }
type SignedData struct {
	Version          int                        ``                          // CMSVersion ::= INTEGER { v0(0), v1(1), v2(2), v3(3), v4(4), v5(5) }
	DigestAlgorithms []pkix.AlgorithmIdentifier `asn1:"set"`                //DigestAlgorithmIdentifiers ::= SET OF DigestAlgorithmIdentifier //DigestAlgorithmIdentifier ::= AlgorithmIdentifier
	EncapContentInfo EncapsulatedContentInfo    ``                          //
	Certificates     []asn1.RawValue            `asn1:"optional,set,tag:0"` // CertificateSet ::= SET OF CertificateChoices
	CRLs             []RevocationInfoChoice     `asn1:"optional,set,tag:1"` // RevocationInfoChoices ::= SET OF RevocationInfoChoice
	SignerInfos      []SignerInfo               `asn1:"set"`                // SignerInfos ::= SET OF SignerInfo
}

// CertificateChoices ::= CHOICE {
//   certificate Certificate,
//   extendedCertificate [0] IMPLICIT ExtendedCertificate, -- Obsolete
//   v1AttrCert [1] IMPLICIT AttributeCertificateV1,       -- Obsolete
//   v2AttrCert [2] IMPLICIT AttributeCertificateV2,
//   other [3] IMPLICIT OtherCertificateFormat }
type CertificateChoices struct {
	Cert       x509.Certificate       `asn1:"optional"`
	V2AttrCert asn1.RawValue          `asn1:"optional,tag:2"`
	Other      OtherCertificateFormat `asn1:"optional,tag:3"`
}

// OtherCertificateFormat ::= SEQUENCE {
//   otherCertFormat OBJECT IDENTIFIER,
//   otherCert ANY DEFINED BY otherCertFormat }
type OtherCertificateFormat struct {
	OtherCertFormat asn1.ObjectIdentifier
	OtherCert       asn1.RawValue
}

// RevocationInfoChoice ::= CHOICE {
//   crl CertificateList,
//   other [1] IMPLICIT OtherRevocationInfoFormat }
type RevocationInfoChoice struct {
	Crl   pkix.CertificateList      `asn1:"optional"`
	Other OtherRevocationInfoFormat `asn1:"optional,tag:1"`
}

// OtherRevocationInfoFormat ::= SEQUENCE {
//   otherRevInfoFormat OBJECT IDENTIFIER,
//   otherRevInfo ANY DEFINED BY otherRevInfoFormat }
type OtherRevocationInfoFormat struct {
	OtherRevInfoFormat asn1.ObjectIdentifier
	OtherRevInfo       asn1.RawValue
}

// NewSignedData creates a new SignedData.
func NewSignedData(eci EncapsulatedContentInfo) (*SignedData, error) {
	// The version is picked based on which CMS features are used. We only use
	// version 1 features, except for supporting non-data econtent.
	version := 1
	if !eci.IsTypeData() {
		version = 3
	}

	return &SignedData{
		Version:          version,
		DigestAlgorithms: []pkix.AlgorithmIdentifier{},
		EncapContentInfo: eci,
		SignerInfos:      []SignerInfo{},
	}, nil
}

// AddSignerInfo adds a SignerInfo to the SignedData.
func (sd *SignedData) AddSignerInfo(keypPair tls.Certificate, attrs []Attribute) (err error) {

	for _, cert := range keypPair.Certificate {
		if err = sd.AddCertificate(cert); err != nil {
			return
		}
	}

	signer := keypPair.PrivateKey.(crypto.Signer)

	cert := keypPair.Leaf

	ias, err := NewIssuerAndSerialNumber(cert)
	if err != nil {
		return err
	}

	sid := SignerIdentifier{ias, nil}

	var signerOpts crypto.SignerOpts
	digestAlgorithm := digestAlgorithmForPublicKey(cert.PublicKey)
	signatureAlgorithm, ok := oid.PublicKeyAlgorithmToSignatureAlgorithm[keypPair.Leaf.PublicKeyAlgorithm]
	if isRSAPSS(cert) {
		h := oid.DigestAlgorithmToHash[digestAlgorithm.Algorithm.String()]
		signatureAlgorithm, signerOpts, err = newPSS(h, cert.PublicKey.(*rsa.PublicKey))
	}
	if !ok {
		return errors.New("unsupported certificate public key algorithm")
	}

	si := SignerInfo{
		Version:            1,
		SID:                sid,
		DigestAlgorithm:    digestAlgorithm,
		SignedAttrs:        nil,
		SignatureAlgorithm: signatureAlgorithm,
		Signature:          nil,
		UnsignedAttrs:      nil,
	}

	// Get the message
	content := sd.EncapContentInfo.EContent
	if err != nil {
		return err
	}
	if content == nil {
		return errors.New("already detached")
	}

	// Digest the message.
	hash, err := si.Hash()
	if err != nil {
		return err
	}

	if !isRSAPSS(cert) {
		signerOpts = hash
	}

	md := hash.New()
	if _, err = md.Write(content); err != nil {
		return err
	}

	// Build our SignedAttributes
	mdAttr, err := NewAttribute(oid.AttributeMessageDigest, md.Sum(nil))
	if err != nil {
		return err
	}
	ctAttr, err := NewAttribute(oid.AttributeContentType, sd.EncapContentInfo.EContentType)
	if err != nil {
		return err
	}
	sTAttr, err := NewAttribute(oid.AttributeSigningTime, time.Now())
	if err != nil {
		return err
	}
	si.SignedAttrs = append(si.SignedAttrs, mdAttr, ctAttr, sTAttr)
	si.SignedAttrs = append(si.SignedAttrs, attrs...)

	sm, err := asn.MarshalWithParams(si.SignedAttrs, `set`)
	if err != nil {
		return err
	}

	smd := hash.New()
	if _, errr := smd.Write(sm); errr != nil {
		return errr
	}
	if si.Signature, err = signer.Sign(rand.Reader, smd.Sum(nil), signerOpts); err != nil {
		return err
	}

	sd.addDigestAlgorithm(si.DigestAlgorithm)

	sd.SignerInfos = append(sd.SignerInfos, si)

	return nil
}

// algorithmsForPublicKey takes an opinionated stance on what algorithms to use
// for the given public key.
func digestAlgorithmForPublicKey(pub crypto.PublicKey) pkix.AlgorithmIdentifier {
	if ecPub, ok := pub.(*ecdsa.PublicKey); ok {
		switch ecPub.Curve {
		case elliptic.P384():
			return pkix.AlgorithmIdentifier{Algorithm: oid.DigestAlgorithmSHA384}
		case elliptic.P521():
			return pkix.AlgorithmIdentifier{Algorithm: oid.DigestAlgorithmSHA512}
		}
	}

	return pkix.AlgorithmIdentifier{Algorithm: oid.DigestAlgorithmSHA256}
}

// ClearCertificates removes all certificates.
func (sd *SignedData) ClearCertificates() {
	sd.Certificates = []asn1.RawValue{}
}

// AddCertificate adds a *x509.Certificate.
func (sd *SignedData) AddCertificate(cert []byte) error {
	for _, existing := range sd.Certificates {
		if bytes.Equal(existing.Bytes, cert) {
			return errors.New("certificate already added")
		}
	}

	var rv asn1.RawValue
	if _, err := asn.Unmarshal(cert, &rv); err != nil {
		return err
	}

	sd.Certificates = append(sd.Certificates, rv)

	return nil
}

// addDigestAlgorithm adds a new AlgorithmIdentifier if it doesn't exist yet.
func (sd *SignedData) addDigestAlgorithm(algo pkix.AlgorithmIdentifier) {
	for _, existing := range sd.DigestAlgorithms {
		if existing.Algorithm.Equal(algo.Algorithm) {
			return
		}
	}

	sd.DigestAlgorithms = append(sd.DigestAlgorithms, algo)
}

// X509Certificates gets the certificates, assuming that they're X.509 encoded.
func (sd *SignedData) X509Certificates() (map[string]*x509.Certificate, error) {
	// Certificates field is optional. Handle missing value.
	if sd.Certificates == nil {
		return nil, nil
	}

	certs := map[string]*x509.Certificate{}

	// Empty set
	if len(sd.Certificates) == 0 {
		return certs, nil
	}

	for _, raw := range sd.Certificates {
		if raw.Class != asn1.ClassUniversal || raw.Tag != asn1.TagSequence {
			return nil, ErrUnsupported
		}

		x509, err := x509.ParseCertificate(raw.FullBytes)
		if err != nil {
			return nil, err
		}
		iasString, err := IASstring(x509)
		certs[iasString] = x509
		if err != nil {
			return nil, err
		}

	}
	return certs, nil
}

// ContentInfo returns the SignedData wrapped in a ContentInfo packet.
func (sd *SignedData) ContentInfo() (ContentInfo, error) {
	var nilCI ContentInfo

	der, err := asn.Marshal(*sd)
	if err != nil {
		return nilCI, err
	}

	return ContentInfo{
		ContentType: oid.SignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			Bytes:      der,
			IsCompound: true,
		},
	}, nil

}

// Verify checks the signature
func (sd *SignedData) Verify(Opts x509.VerifyOptions, detached []byte) (chains [][][]*x509.Certificate, err error) {
	certs, _ := sd.X509Certificates()

	opts := Opts

	for _, c := range certs {
		opts.Intermediates.AddCert(c)
		intermediates, fetchErr := fetchIntermediates(c.IssuingCertificateURL)
		for _, e := range fetchErr {
			fmt.Printf("Error while fetching intermediates: %s\n", e)
		}

		for _, i := range intermediates {
			opts.Intermediates.AddCert(i)
		}
	}

	eContent := detached
	if eContent == nil {
		eContent = sd.EncapContentInfo.EContent
	}

	for _, signer := range sd.SignerInfos {
		//Find and check signer Certificate:
		sidxxx, _ := signer.SID.IAS.RawValue()
		sid := fmt.Sprintf("%x", sidxxx.Bytes)

		cert, exist := certs[sid]

		if !exist {
			err = errors.New("Could not find a Certificate for signer with sid : " + sid)
			return
		}

		var signingTime time.Time
		signingTime, err = signer.GetSigningTimeAttribute()
		if err != nil {
			opts.CurrentTime = time.Now()
		}
		opts.CurrentTime = signingTime

		var chain [][]*x509.Certificate
		chain, err = cert.Verify(opts)
		if err != nil {
			//	return
		}

		signedMessage := eContent
		if signer.SignedAttrs != nil {

			//Hash message:
			var hash crypto.Hash
			hash, err = signer.Hash()
			if err != nil {
				return nil, err
			}
			md := hash.New()

			_, err = md.Write(eContent)
			if err != nil {
				return nil, err
			}
			h := md.Sum(nil)

			var messageDigestAttr []byte
			messageDigestAttr, err = signer.GetMessageDigestAttribute()
			if err != nil {
				return
			}

			if !bytes.Equal(messageDigestAttr, h) {
				err = errors.New("Signed hash does not match the hash of the message")
				return
			}

			signedMessage, err = asn.MarshalWithParams(signer.SignedAttrs, `set`)
			if err != nil {
				return
			}
		}
		var sigAlg x509.SignatureAlgorithm
		sigAlg, err = signer.X509SignatureAlgorithm()
		if err != nil {
			return
		}
		switch signer.SignatureAlgorithm.Algorithm.String() {
		case oid.SignatureAlgorithmRSASSAPSS.String():
		default:
			err = cert.CheckSignature(sigAlg, signedMessage, signer.Signature)
		}
		if err != nil {
			return
		}
		chains = append(chains, chain)
	}

	return
}

func fetchIntermediates(urls []string) (certificates []*x509.Certificate, errs []error) {
	for _, url := range urls {
		var resp *http.Response
		resp, err := http.Get(url)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		defer resp.Body.Close()

		issuerBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		issuerCert, err := x509.ParseCertificate(issuerBytes)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		//Prevent infinite loop
		if len(certificates) > 50 {
			err = errors.New("To many issuers")
			errs = append(errs, err)
			return
		}
		certificates = append(certificates, issuerCert)

		//Recusively fetch issuers
		issuers, fetchErrs := fetchIntermediates(issuerCert.IssuingCertificateURL)
		certificates = append(certificates, issuers...)
		errs = append(errs, fetchErrs...)
	}
	return
}
