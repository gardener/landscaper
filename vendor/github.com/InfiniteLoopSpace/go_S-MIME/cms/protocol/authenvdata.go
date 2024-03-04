package protocol

import (
	"crypto/tls"
	"encoding/asn1"
	"log"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

//AuthEnvelopedData ::= SEQUENCE {
//	version CMSVersion,
//	originatorInfo [0] IMPLICIT OriginatorInfo OPTIONAL,
//	recipientInfos RecipientInfos,
//	authEncryptedContentInfo EncryptedContentInfo,
///	authAttrs [1] IMPLICIT AuthAttributes OPTIONAL,
//	mac MessageAuthenticationCode,
//	unauthAttrs [2] IMPLICIT UnauthAttributes OPTIONAL }
//https://tools.ietf.org/html/rfc5083##section-2.1
type AuthEnvelopedData struct {
	Version        int
	OriginatorInfo asn1.RawValue   `asn1:"optional,tag:0"`
	RecipientInfos []RecipientInfo `asn1:"set,choice"`
	AECI           EncryptedContentInfo
	AauthAttrs     []Attribute `asn1:"set,optional,tag:1"`
	MAC            []byte
	UnAauthAttrs   []Attribute `asn1:"set,optional,tag:2"`
}

// Decrypt decrypts AuthEnvelopedData and returns the plaintext.
func (ed *AuthEnvelopedData) Decrypt(keyPair []tls.Certificate) (plain []byte, err error) {

	// Find the right key
	var key []byte
	for i := range keyPair {
		key, err = ed.decryptKey(keyPair[i])
		switch err {
		case ErrNoKeyFound:
			continue
		case nil:
			break
		default:
			return
		}
	}

	encAlg := &oid.EncryptionAlgorithm{
		Key:                                  key,
		ContentEncryptionAlgorithmIdentifier: ed.AECI.ContentEncryptionAlgorithm,
	}
	encAlg.MAC = ed.MAC

	plain, err = encAlg.Decrypt(ed.AECI.EContent)

	return
}

func (ed *AuthEnvelopedData) decryptKey(keyPair tls.Certificate) (key []byte, err error) {

	for i := range ed.RecipientInfos {

		key, err = ed.RecipientInfos[i].decryptKey(keyPair)
		if key != nil {
			return
		}
	}
	return nil, ErrNoKeyFound
}

// NewAuthEnvelopedData creates AuthEnvelopedData from an EncryptedContentInfo with mac and given RecipientInfos.
func NewAuthEnvelopedData(eci *EncryptedContentInfo, reciInfos []RecipientInfo, mac []byte) AuthEnvelopedData {
	version := 0

	ed := AuthEnvelopedData{
		Version:        version,
		RecipientInfos: reciInfos,
		AECI:           *eci,
		MAC:            mac,
	}

	return ed
}

func authcontentInfo(ed AuthEnvelopedData) (ci ContentInfo, err error) {

	der, err := asn.Marshal(ed)
	if err != nil {
		return
	}

	ci = ContentInfo{
		ContentType: oid.AuthEnvelopedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			Bytes:      der,
			IsCompound: true,
		},
	}

	return
}

// ContentInfo marshals AuthEnvelopedData and returns ContentInfo.
func (ed AuthEnvelopedData) ContentInfo() (ContentInfo, error) {
	nilCI := *new(ContentInfo)

	der, err := asn.Marshal(ed)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		return nilCI, err
	}

	return ContentInfo{
		ContentType: oid.AuthEnvelopedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			Bytes:      der,
			IsCompound: true,
		},
	}, nil

}

// AuthEnvelopedDataContent unmarshals ContentInfo and returns AuthEnvelopedData if
// content type is AuthEnvelopedData.
func (ci ContentInfo) AuthEnvelopedDataContent() (*AuthEnvelopedData, error) {
	if !ci.ContentType.Equal(oid.AuthEnvelopedData) {
		return nil, ErrWrongType
	}

	ed := new(AuthEnvelopedData)
	if rest, err := asn.Unmarshal(ci.Content.Bytes, ed); err != nil {
		return nil, err
	} else if len(rest) > 0 {
		return nil, ErrTrailingData
	}

	return ed, nil
}
