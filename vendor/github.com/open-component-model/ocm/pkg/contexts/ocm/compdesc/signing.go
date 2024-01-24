// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package compdesc

import (
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/signutils"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	KIND_HASH_ALGORITHM   = signutils.KIND_HASH_ALGORITHM
	KIND_SIGN_ALGORITHM   = signutils.KIND_SIGN_ALGORITHM
	KIND_NORM_ALGORITHM   = signutils.KIND_NORM_ALGORITHM
	KIND_VERIFY_ALGORITHM = signutils.KIND_VERIFY_ALGORITHM
	KIND_PUBLIC_KEY       = signutils.KIND_PUBLIC_KEY
	KIND_PRIVATE_KEY      = signutils.KIND_PRIVATE_KEY
	KIND_SIGNATURE        = signutils.KIND_SIGNATURE
	KIND_DIGEST           = signutils.KIND_DIGEST
)

// IsNormalizeable checks if componentReferences and resources contain digest.
// Resources are allowed to omit the digest, if res.access.type == None or res.access == nil.
// Does NOT verify if the digests are correct.
func (cd *ComponentDescriptor) IsNormalizeable() error {
	// check for digests on component references
	for _, reference := range cd.References {
		if reference.Digest == nil || reference.Digest.HashAlgorithm == "" || reference.Digest.NormalisationAlgorithm == "" || reference.Digest.Value == "" {
			return fmt.Errorf("missing digest in componentReference for %s:%s", reference.Name, reference.Version)
		}
	}
	for _, res := range cd.Resources {
		if (res.Access != nil && res.Access.GetType() != "None") && res.Digest == nil {
			return fmt.Errorf("missing digest in resource for %s:%s", res.Name, res.Version)
		}
		if (res.Access == nil || res.Access.GetType() == "None") && res.Digest != nil {
			return fmt.Errorf("digest for resource with empty (None) access not allowed %s:%s", res.Name, res.Version)
		}
	}
	return nil
}

// Hash return the hash for the component-descriptor, if it is normalizeable
// (= componentReferences and resources contain digest field).
func Hash(cd *ComponentDescriptor, normAlgo string, hash hash.Hash) (string, error) {
	if hash == nil {
		return metav1.NoDigest, nil
	}

	normalized, err := Normalize(cd, normAlgo)
	if err != nil {
		return "", fmt.Errorf("failed normalising component descriptor %w", err)
	}
	// fmt.Printf("NORM %s:%s: %s\n", cd.Name, cd.Version, string(normalized))
	hash.Reset()
	if _, err = hash.Write(normalized); err != nil {
		return "", fmt.Errorf("failed hashing the normalisedComponentDescriptorJson: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type CompDescDigest struct {
	normAlgo   string
	hashAlgo   string
	normalized []byte
	digest     string
}

type CompDescDigests struct {
	cd      *ComponentDescriptor
	digests []*CompDescDigest
}

func NewCompDescDigests(cd *ComponentDescriptor) *CompDescDigests {
	return &CompDescDigests{
		cd: cd,
	}
}

func (d *CompDescDigests) Descriptor() *ComponentDescriptor {
	return d.cd
}

func (d *CompDescDigests) Get(normAlgo string, hasher signing.Hasher) ([]byte, string, error) {
	var normalized []byte

	for _, e := range d.digests {
		if e.normAlgo == normAlgo {
			normalized = e.normalized
			if e.hashAlgo == hasher.Algorithm() {
				return e.normalized, e.digest, nil
			}
		}
	}

	var err error
	if normalized == nil {
		normalized, err = Normalize(d.cd, normAlgo)
		if err != nil {
			return nil, "", fmt.Errorf("failed normalising component descriptor %w", err)
		}
	}
	hash := hasher.Create()
	if _, err = hash.Write(normalized); err != nil {
		return nil, "", fmt.Errorf("failed hashing the normalisedComponentDescriptorJson: %w", err)
	}

	e := &CompDescDigest{
		normAlgo:   normAlgo,
		hashAlgo:   hasher.Algorithm(),
		normalized: normalized,
		digest:     hex.EncodeToString(hash.Sum(nil)),
	}
	d.digests = append(d.digests, e)
	return normalized, e.digest, nil
}

func NormHash(cd *ComponentDescriptor, normAlgo string, hash hash.Hash) ([]byte, string, error) {
	if hash == nil {
		return nil, metav1.NoDigest, nil
	}

	normalized, err := Normalize(cd, normAlgo)
	if err != nil {
		return nil, "", fmt.Errorf("failed normalising component descriptor %w", err)
	}
	hash.Reset()
	if _, err = hash.Write(normalized); err != nil {
		return nil, "", fmt.Errorf("failed hashing the normalisedComponentDescriptorJson: %w", err)
	}

	return normalized, hex.EncodeToString(hash.Sum(nil)), nil
}

// Sign signs the given component-descriptor with the signer.
// The component-descriptor has to contain digests for componentReferences and resources.
func Sign(cctx credentials.Context, cd *ComponentDescriptor, privateKey signutils.GenericPrivateKey, signer signing.Signer, hasher signing.Hasher, signatureName, issuer string) error {
	digest, err := Hash(cd, JsonNormalisationV1, hasher.Create())
	if err != nil {
		return fmt.Errorf("failed getting hash for cd: %w", err)
	}

	iss, err := signutils.ParseDN(issuer)
	if err != nil {
		return err
	}

	sctx := &signing.DefaultSigningContext{
		Hash:       hasher.Crypto(),
		PrivateKey: privateKey,
		PublicKey:  nil,
		RootCerts:  nil,
		Issuer:     iss,
	}

	signature, err := signer.Sign(cctx, digest, sctx)
	if err != nil {
		return fmt.Errorf("failed signing hash of normalised component descriptor, %w", err)
	}
	signature.Issuer = issuer
	cd.Signatures = append(cd.Signatures, metav1.Signature{
		Name: signatureName,
		Digest: metav1.DigestSpec{
			HashAlgorithm:          hasher.Algorithm(),
			NormalisationAlgorithm: JsonNormalisationV1,
			Value:                  digest,
		},
		Signature: metav1.SignatureSpec{
			Algorithm: signature.Algorithm,
			Value:     signature.Value,
			MediaType: signature.MediaType,
			Issuer:    signature.Issuer,
		},
	})
	return nil
}

// Verify verifies the signature (selected by signatureName) and hash of the component-descriptor (as specified in the signature).
// Does NOT resolve resources or referenced component-descriptors.
// Returns error if verification fails.
func Verify(cd *ComponentDescriptor, registry signing.Registry, signatureName string, rootCA ...signutils.GenericCertificatePool) error {
	// find matching signature
	matchingSignature := cd.SelectSignatureByName(signatureName)
	if matchingSignature == nil {
		return errors.ErrNotFound(KIND_SIGNATURE, signatureName)
	}
	verifier := registry.GetVerifier(matchingSignature.Signature.Algorithm)
	if verifier == nil {
		return errors.ErrUnknown(KIND_SIGN_ALGORITHM, matchingSignature.Signature.Algorithm)
	}
	publicKey := registry.GetPublicKey(signatureName)
	if publicKey == nil {
		return errors.ErrNotFound(KIND_PUBLIC_KEY, signatureName)
	}
	hasher := registry.GetHasher(matchingSignature.Digest.HashAlgorithm)
	if hasher == nil {
		return errors.ErrUnknown(KIND_HASH_ALGORITHM, matchingSignature.Digest.HashAlgorithm)
	}
	// Verify author of signature
	sctx := &signing.DefaultSigningContext{
		Hash:      hasher.Crypto(),
		PublicKey: publicKey,
		RootCerts: utils.Optional(rootCA),
		Issuer:    registry.GetIssuer(signatureName),
	}
	err := verifier.Verify(matchingSignature.Digest.Value, matchingSignature.ConvertToSigning(), sctx)
	if err != nil {
		return fmt.Errorf("failed verifying: %w", err)
	}

	hash := hasher.Create()
	// Verify normalised cd to given (and verified) hash
	calculatedDigest, err := Hash(cd, matchingSignature.Digest.NormalisationAlgorithm, hash)
	if err != nil {
		return fmt.Errorf("failed hashing cd %s:%s: %w", cd.Name, cd.Version, err)
	}

	if calculatedDigest != matchingSignature.Digest.Value {
		return fmt.Errorf("normalised component-descriptor does not match hash from signature")
	}

	return nil
}

// SelectSignatureByName returns the Signature (Digest and SigantureSpec) matching the given name.
func (cd *ComponentDescriptor) SelectSignatureByName(signatureName string) *metav1.Signature {
	for _, signature := range cd.Signatures {
		if signature.Name == signatureName {
			return &signature
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (cd *ComponentDescriptor) HasResourceDigests() bool {
	return cd.Resources.HaveDigests()
}
