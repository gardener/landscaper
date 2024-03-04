package protocol

import (
	"crypto/x509/pkix"
	"encoding/asn1"

	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

//EncryptedContentInfo ::= SEQUENCE {
//	contentType ContentType,
//	contentEncryptionAlgorithm ContentEncryptionAlgorithmIdentifier,
//	encryptedContent [0] IMPLICIT EncryptedContent OPTIONAL }
type EncryptedContentInfo struct {
	EContentType               asn1.ObjectIdentifier
	ContentEncryptionAlgorithm pkix.AlgorithmIdentifier
	EContent                   []byte `asn1:"optional,implicit,tag:0"`
}

// NewEncryptedContentInfo encrypts the conent with the contentEncryptionAlgorithm and retuns
// the EncryptedContentInfo, the key and the MAC.
func NewEncryptedContentInfo(contentType asn1.ObjectIdentifier, contentEncryptionAlg asn1.ObjectIdentifier, content []byte) (eci EncryptedContentInfo, key, mac []byte, err error) {

	encAlg := &oid.EncryptionAlgorithm{
		EncryptionAlgorithmIdentifier: contentEncryptionAlg,
	}

	ciphertext, err := encAlg.Encrypt(content)
	if err != nil {
		return
	}

	eci = EncryptedContentInfo{
		EContentType:               contentType,
		ContentEncryptionAlgorithm: encAlg.ContentEncryptionAlgorithmIdentifier,
		EContent:                   ciphertext,
	}

	return eci, encAlg.Key, encAlg.MAC, nil
}
