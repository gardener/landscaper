// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/repositories/artifactset"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/localblob"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha256"
	"github.com/open-component-model/ocm/pkg/signing/hasher/sha512"
	"github.com/open-component-model/ocm/pkg/utils"
)

const OciArtifactDigestV1 string = "ociArtifactDigest/v1"

const LegacyOciArtifactDigestV1 string = "ociArtefactDigest/v1"

func init() {
	cpi.MustRegisterDigester(New(sha256.Algorithm), "")
	cpi.MustRegisterDigester(New(sha512.Algorithm), "")

	// legacy digester types
	cpi.MustRegisterDigester(New(digest.SHA256.String(), OciArtifactDigestV1), "")
	cpi.MustRegisterDigester(New(digest.SHA512.String(), OciArtifactDigestV1), "")

	cpi.MustRegisterDigester(New(digest.SHA256.String(), LegacyOciArtifactDigestV1), "")
	cpi.MustRegisterDigester(New(digest.SHA512.String(), LegacyOciArtifactDigestV1), "")
}

func New(algo string, ts ...string) cpi.BlobDigester {
	norm := utils.OptionalDefaulted(OciArtifactDigestV1, ts...)
	return &Digester{
		cpi.DigesterType{
			HashAlgorithm:          algo,
			NormalizationAlgorithm: norm,
		},
	}
}

type Digester struct {
	typ cpi.DigesterType
}

var _ cpi.BlobDigester = (*Digester)(nil)

func (d *Digester) GetType() cpi.DigesterType {
	return d.typ
}

func (d *Digester) DetermineDigest(reftyp string, acc cpi.AccessMethod, preferred signing.Hasher) (*cpi.DigestDescriptor, error) {
	if acc.GetKind() == localblob.Type {
		mime := acc.MimeType()
		if !artdesc.IsOCIMediaType(mime) {
			return nil, nil
		}
		r, err := acc.Reader()
		if err != nil {
			return nil, err
		}
		defer r.Close()

		var reader io.Reader = r
		if strings.HasSuffix(mime, "+gzip") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				return nil, err
			}
		}
		tr := tar.NewReader(reader)

		var desc *cpi.DigestDescriptor
		oci := false
		layout := false
		for {
			header, err := tr.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					if oci {
						if layout {
							return desc, nil
						} else {
							err = fmt.Errorf("oci-layout not found")
						}
					} else {
						err = fmt.Errorf("descriptor not found in archive")
					}
				}
				return nil, errors.ErrInvalidWrap(err, "artifact archive")
			}

			switch header.Typeflag {
			case tar.TypeDir:
			case tar.TypeReg:
				switch header.Name {
				case artifactset.OCILayouFileName:
					layout = true
				case artifactset.OCIArtifactSetDescriptorFileName:
					oci = true
					fallthrough
				case artifactset.ArtifactSetDescriptorFileName:
					data, err := io.ReadAll(tr)
					if err != nil {
						return nil, fmt.Errorf("unable to read descriptor from archive: %w", err)
					}
					index, err := artdesc.DecodeIndex(data)
					if err != nil {
						return nil, err
					}
					if index == nil {
						return nil, fmt.Errorf("no main artifact found")
					}
					main := artifactset.RetrieveMainArtifact(index.Annotations)
					if main == "" {
						return nil, fmt.Errorf("no main artifact found")
					}
					if d.GetType().HashAlgorithm != signing.NormalizeHashAlgorithm(string(digest.Digest(main).Algorithm())) {
						return nil, nil
					}
					desc = cpi.NewDigestDescriptor(digest.Digest(main).Hex(), d.GetType())
					if !oci {
						return desc, nil
					}
				}
			}
		}
		// not reached (endless for)
	}
	if ociartifact.Is(acc.AccessSpec()) {
		var (
			dig digest.Digest
			err error
		)

		// first: check for error providing interface
		if s, ok := acc.(DigestSource); ok {
			dig, err = s.GetDigest()
		} else {
			// second: fallback to standard digest interface
			dig = acc.(accessio.DigestSource).Digest()
		}

		if dig != "" {
			if d.GetType().HashAlgorithm != signing.NormalizeHashAlgorithm(dig.Algorithm().String()) {
				return nil, nil
			}
			return cpi.NewDigestDescriptor(dig.Hex(), d.GetType()), nil
		}
		return nil, errors.NewEf(err, "cannot determine digest")
	}
	return nil, nil
}

type DigestSource interface {
	GetDigest() (digest.Digest, error)
}
