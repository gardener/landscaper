// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package artifactset

import (
	"fmt"

	. "github.com/open-component-model/ocm/pkg/finalizer"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/transfer"
	"github.com/open-component-model/ocm/pkg/errors"
)

const SynthesizedBlobFormat = "+tar+gzip"

type ArtifactBlob interface {
	blobaccess.BlobAccess
}

type Producer func(set *ArtifactSet) (string, error)

func SythesizeArtifactSet(producer Producer) (ArtifactBlob, error) {
	temp, err := blobaccess.NewTempFile("", "artifactblob*.tgz")
	if err != nil {
		return nil, err
	}
	defer temp.Close()

	set, err := Create(accessobj.ACC_CREATE, "", 0o600, accessio.File(temp.Writer().(vfs.File)), accessobj.FormatTGZ)
	if err != nil {
		return nil, err
	}
	mime, err := producer(set)
	err2 := set.Close()
	if err != nil {
		return nil, err
	}
	if err2 != nil {
		return nil, err2
	}

	return temp.AsBlob(MediaType(mime)), nil
}

func MediaType(mime string) string {
	return artdesc.ToContentMediaType(mime) + SynthesizedBlobFormat
}

func TransferArtifact(art cpi.ArtifactAccess, set cpi.ArtifactSink, tags ...string) error {
	return transfer.TransferArtifact(art, set, tags...)
}

type ArtifactModifier func(access cpi.ArtifactAccess) error

// SynthesizeArtifactBlob synthesizes an artifact blob incorporating all side artifacts.
// To support extensions like cosign, we need the namespace access her to find
// additionally objects associated by tags.
func SynthesizeArtifactBlob(ns cpi.NamespaceAccess, ref string, mod ...ArtifactModifier) (ArtifactBlob, error) {
	art, err := ns.GetArtifact(ref)
	if err != nil {
		return nil, GetArtifactError{Original: err, Ref: ref}
	}
	defer art.Close()

	for _, m := range mod {
		err = m(art)
		if err != nil {
			return nil, err
		}
	}
	return SynthesizeArtifactBlobForArtifact(art, ref)
}

func SynthesizeArtifactBlobForArtifact(art cpi.ArtifactAccess, ref string) (ArtifactBlob, error) {
	blob, err := art.Blob()
	if err != nil {
		return nil, err
	}
	digest := blob.Digest()

	return SythesizeArtifactSet(func(set *ArtifactSet) (string, error) {
		err = TransferArtifact(art, set)
		if err != nil {
			return "", fmt.Errorf("failed to transfer artifact: %w", err)
		}

		if ok, _ := artdesc.IsDigest(ref); !ok {
			err = set.AddTags(digest, ref)
			if err != nil {
				return "", fmt.Errorf("failed to add tag: %w", err)
			}
		}

		set.Annotate(MAINARTIFACT_ANNOTATION, digest.String())

		return blob.MimeType(), nil
	})
}

// ArtifactFactory add an artifact to the given set and provides descriptor metadata.
type ArtifactFactory func(set *ArtifactSet) (digest.Digest, string, error)

// ArtifactIterator provides a sequence of artifact factories by successive calls.
// The sequence is finished if nil is returned for the factory.
type ArtifactIterator func() (ArtifactFactory, bool, error)

// ArtifactFeedback is called after an artifact has successfully be added.
type ArtifactFeedback func(blob blobaccess.BlobAccess, art cpi.ArtifactAccess) error

// ArtifactTransferCreator provides an ArtifactFactory transferring the given artifact.
func ArtifactTransferCreator(art cpi.ArtifactAccess, finalizer *Finalizer, feedback ...ArtifactFeedback) ArtifactFactory {
	return func(set *ArtifactSet) (digest.Digest, string, error) {
		var f Finalizer
		defer f.Finalize()

		f.Include(finalizer)

		blob, err := art.Blob()
		if err != nil {
			return "", "", errors.Wrapf(err, "cannot access artifact manifest blob")
		}
		f.Close(blob)

		err = TransferArtifact(art, set)
		if err != nil {
			return "", "", fmt.Errorf("failed to transfer artifact: %w", err)
		}

		list := errors.ErrListf("add artifact")
		for _, fb := range feedback {
			list.Add(fb(blob, art))
		}
		return blob.Digest(), blob.MimeType(), list.Result()
	}
}

// SynthesizeArtifactBlobFor synthesizes an artifact blob incorporating all artifacts
// provided ba a factory.
func SynthesizeArtifactBlobFor(tag string, iter ArtifactIterator) (ArtifactBlob, error) {
	return SythesizeArtifactSet(func(set *ArtifactSet) (string, error) {
		mime := artdesc.MediaTypeImageManifest
		for {
			art, main, err := iter()
			if err != nil || art == nil {
				return mime, err
			}

			digest, _mime, err := art(set)
			if err != nil {
				return "", err
			}
			if main {
				if mime != "" {
					mime = _mime
				}
				set.Annotate(MAINARTIFACT_ANNOTATION, digest.String())
				if tag != "" {
					err = set.AddTags(digest, tag)
					if err != nil {
						return "", fmt.Errorf("failed to add tag: %w", err)
					}
				}
			}
		}
	})
}
