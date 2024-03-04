// Package oid contains OIDs that are used by other packages in this repository.
package oid

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
)

// Content type OIDs
var (
	Data               = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	SignedData         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	EnvelopedData      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 3}
	AuthEnvelopedData  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 23}
	TSTInfo            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	ContentTypeTSTInfo = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
)

// Attribute OIDs
var (
	AttributeContentType    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	AttributeMessageDigest  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	AttributeSigningTime    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
	AttributeTimeStampToken = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}
)

// Signature Algorithm  OIDs
var (
	SignatureAlgorithmRSA             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	SignatureAlgorithmRSASSAPSS       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	SignatureAlgorithmECDSA           = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	SignatureAlgorithmECDSAwithSHA1   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1}
	SignatureAlgorithmECDSAwithSHA224 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 1}
	SignatureAlgorithmECDSAwithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	SignatureAlgorithmECDSAwithSHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	SignatureAlgorithmECDSAwithSHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
)

// Public Key Encryption OIDs
var (
	EncryptionAlgorithmRSA       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	EncryptionAlgorithmRSAESOAEP = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 7}
)

// Digest Algorithm OIDs
var (
	DigestAlgorithmSHA1   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	DigestAlgorithmMD5    = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 5}
	DigestAlgorithmSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	DigestAlgorithmSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	DigestAlgorithmSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
)

// X.509 extensions
var (
	SubjectKeyIdentifier = asn1.ObjectIdentifier{2, 5, 29, 14}
)

// Elliptic curve public key OID
var (
	ECPublicKey = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
)

// DH Key Derivation Schemes OIDs
var (
	DHSinglePassstdDHsha1kdfscheme   = asn1.ObjectIdentifier{1, 3, 133, 16, 840, 63, 0, 2}
	DHSinglePassstdDHsha224kdfscheme = asn1.ObjectIdentifier{1, 3, 132, 1, 11, 0}
	DHSinglePassstdDHsha256kdfscheme = asn1.ObjectIdentifier{1, 3, 132, 1, 11, 1}
	DHSinglePassstdDHsha384kdfscheme = asn1.ObjectIdentifier{1, 3, 132, 1, 11, 2}
	DHSinglePassstdDHsha512kdfscheme = asn1.ObjectIdentifier{1, 3, 132, 1, 11, 3}
)

// Key wrap algorithm OIDs
var (
	AES128Wrap = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 5}
	AES192Wrap = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 25}
	AES256Wrap = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 45}
)

// DigestAlgorithmToHash maps digest OIDs to crypto.Hash values.
var DigestAlgorithmToHash = map[string]crypto.Hash{
	DigestAlgorithmSHA1.String():   crypto.SHA1,
	DigestAlgorithmMD5.String():    crypto.MD5,
	DigestAlgorithmSHA256.String(): crypto.SHA256,
	DigestAlgorithmSHA384.String(): crypto.SHA384,
	DigestAlgorithmSHA512.String(): crypto.SHA512,
}

// HashToDigestAlgorithm maps crypto.Hash values to digest OIDs.
var HashToDigestAlgorithm = map[crypto.Hash]asn1.ObjectIdentifier{
	crypto.SHA1:   DigestAlgorithmSHA1,
	crypto.MD5:    DigestAlgorithmMD5,
	crypto.SHA256: DigestAlgorithmSHA256,
	crypto.SHA384: DigestAlgorithmSHA384,
	crypto.SHA512: DigestAlgorithmSHA512,
}

// SignatureAlgorithmToDigestAlgorithm maps x509.SignatureAlgorithm to
// digestAlgorithm OIDs.
var SignatureAlgorithmToDigestAlgorithm = map[x509.SignatureAlgorithm]asn1.ObjectIdentifier{
	x509.SHA1WithRSA:     DigestAlgorithmSHA1,
	x509.MD5WithRSA:      DigestAlgorithmMD5,
	x509.SHA256WithRSA:   DigestAlgorithmSHA256,
	x509.SHA384WithRSA:   DigestAlgorithmSHA384,
	x509.SHA512WithRSA:   DigestAlgorithmSHA512,
	x509.ECDSAWithSHA1:   DigestAlgorithmSHA1,
	x509.ECDSAWithSHA256: DigestAlgorithmSHA256,
	x509.ECDSAWithSHA384: DigestAlgorithmSHA384,
	x509.ECDSAWithSHA512: DigestAlgorithmSHA512,
}

// SignatureAlgorithmToSignatureAlgorithm maps x509.SignatureAlgorithm to
// signatureAlgorithm OIDs.
var SignatureAlgorithmToSignatureAlgorithm = map[x509.SignatureAlgorithm]asn1.ObjectIdentifier{
	x509.SHA1WithRSA:     SignatureAlgorithmRSA,
	x509.MD5WithRSA:      SignatureAlgorithmRSA,
	x509.SHA256WithRSA:   SignatureAlgorithmRSA,
	x509.SHA384WithRSA:   SignatureAlgorithmRSA,
	x509.SHA512WithRSA:   SignatureAlgorithmRSA,
	x509.ECDSAWithSHA1:   SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA256: SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA384: SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA512: SignatureAlgorithmECDSA,
}

// SignatureAlgorithms maps digest and signature OIDs to
// x509.SignatureAlgorithm values.
var SignatureAlgorithms = map[string]map[string]x509.SignatureAlgorithm{
	SignatureAlgorithmRSA.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA1.String():   x509.SHA1WithRSA,
		DigestAlgorithmMD5.String():    x509.MD5WithRSA,
		DigestAlgorithmSHA256.String(): x509.SHA256WithRSA,
		DigestAlgorithmSHA384.String(): x509.SHA384WithRSA,
		DigestAlgorithmSHA512.String(): x509.SHA512WithRSA,
	},
	SignatureAlgorithmRSASSAPSS.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA256.String(): x509.SHA256WithRSAPSS,
		DigestAlgorithmSHA384.String(): x509.SHA384WithRSAPSS,
		DigestAlgorithmSHA512.String(): x509.SHA512WithRSAPSS,
	},
	SignatureAlgorithmECDSA.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA1.String():   x509.ECDSAWithSHA1,
		DigestAlgorithmSHA256.String(): x509.ECDSAWithSHA256,
		DigestAlgorithmSHA384.String(): x509.ECDSAWithSHA384,
		DigestAlgorithmSHA512.String(): x509.ECDSAWithSHA512,
	},
	SignatureAlgorithmECDSAwithSHA1.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA1.String(): x509.ECDSAWithSHA1,
	},
	SignatureAlgorithmECDSAwithSHA256.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA256.String(): x509.ECDSAWithSHA256,
	},
	SignatureAlgorithmECDSAwithSHA384.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA384.String(): x509.ECDSAWithSHA384,
	},
	SignatureAlgorithmECDSAwithSHA512.String(): map[string]x509.SignatureAlgorithm{
		DigestAlgorithmSHA512.String(): x509.ECDSAWithSHA512,
	},
}

// PublicKeyAlgorithmToSignatureAlgorithm maps certificate public key
// algorithms to CMS signature algorithms.
var PublicKeyAlgorithmToSignatureAlgorithm = map[x509.PublicKeyAlgorithm]pkix.AlgorithmIdentifier{
	x509.RSA:   pkix.AlgorithmIdentifier{Algorithm: SignatureAlgorithmRSA},
	x509.ECDSA: pkix.AlgorithmIdentifier{Algorithm: SignatureAlgorithmECDSA},
}

// PublicKeyAlgorithmToEncrytionAlgorithm maps certificate public key
// algorithms to CMS encryption algorithms.
var PublicKeyAlgorithmToEncrytionAlgorithm = map[x509.PublicKeyAlgorithm]pkix.AlgorithmIdentifier{
	x509.RSA: pkix.AlgorithmIdentifier{Algorithm: EncryptionAlgorithmRSA},
}

// KDFHashAlgorithm key derivation schemes to its hash algorithms
var KDFHashAlgorithm = map[string]crypto.Hash{
	DHSinglePassstdDHsha1kdfscheme.String():   crypto.SHA1,
	DHSinglePassstdDHsha224kdfscheme.String(): crypto.SHA224,
	DHSinglePassstdDHsha256kdfscheme.String(): crypto.SHA256,
	DHSinglePassstdDHsha384kdfscheme.String(): crypto.SHA384,
	DHSinglePassstdDHsha512kdfscheme.String(): crypto.SHA512,
}
