package protocol

import (
	"crypto/tls"
	"encoding/asn1"
	"log"

	asn "github.com/InfiniteLoopSpace/go_S-MIME/asn1"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

//EnvelopedData ::= SEQUENCE {
//	version CMSVersion,
//	originatorInfo [0] IMPLICIT OriginatorInfo OPTIONAL,
//	recipientInfos RecipientInfos,
//	encryptedContentInfo EncryptedContentInfo,
//	unprotectedAttrs [1] IMPLICIT UnprotectedAttributes OPTIONAL }
type EnvelopedData struct {
	Version          int
	OriginatorInfo   asn1.RawValue        `asn1:"optional,tag:0"`
	RecipientInfos   []RecipientInfo      `asn1:"set,choice"`
	ECI              EncryptedContentInfo ``
	UnprotectedAttrs []Attribute          `asn1:"set,optional,tag:1"`
}

// Decrypt decrypts the EnvelopedData with the given keyPair and retuns the plaintext.
func (ed *EnvelopedData) Decrypt(keyPairs []tls.Certificate) (plain []byte, err error) {

	// Find the right key
	var key []byte
	for i := range keyPairs {
		key, err = ed.decryptKey(keyPairs[i])
		switch err {
		case ErrNoKeyFound:
			continue
		case nil:
			break
		default:
			return
		}
	}
	if key == nil {
		return nil, ErrNoKeyFound
	}

	encAlg := &oid.EncryptionAlgorithm{
		Key:                                  key,
		ContentEncryptionAlgorithmIdentifier: ed.ECI.ContentEncryptionAlgorithm,
	}

	plain, err = encAlg.Decrypt(ed.ECI.EContent)

	return
}

func (ed *EnvelopedData) decryptKey(keyPair tls.Certificate) (key []byte, err error) {

	for i := range ed.RecipientInfos {

		key, err = ed.RecipientInfos[i].decryptKey(keyPair)
		if key != nil || err != ErrNoKeyFound {
			return
		}
	}
	return nil, ErrNoKeyFound
}

// EnvelopedDataContent returns EnvelopedData if ContentType is EnvelopedData.
func (ci ContentInfo) EnvelopedDataContent() (*EnvelopedData, error) {
	if !ci.ContentType.Equal(oid.EnvelopedData) {
		return nil, ErrWrongType
	}

	//var Ed interface{}
	ed := new(EnvelopedData)
	if rest, err := asn.Unmarshal(ci.Content.Bytes, ed); err != nil {
		return nil, err
	} else if len(rest) > 0 {
		return nil, ErrTrailingData
	}

	return ed, nil
}

// ContentInfo returns new ContentInfo with ContentType EnvelopedData.
func (ed EnvelopedData) ContentInfo() (ContentInfo, error) {
	nilCI := *new(ContentInfo)

	der, err := asn.Marshal(ed)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		return nilCI, err
	}

	return ContentInfo{
		ContentType: oid.EnvelopedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			Bytes:      der,
			IsCompound: true,
		},
	}, nil

}

// NewEnvelopedData creates a new EnvelopedData from the given data.
func NewEnvelopedData(eci *EncryptedContentInfo, reciInfos []RecipientInfo) EnvelopedData {
	version := 0

	ed := EnvelopedData{
		Version:        version,
		RecipientInfos: reciInfos,
		ECI:            *eci,
	}

	return ed
}
