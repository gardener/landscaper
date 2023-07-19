// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/logging"
)

func TransferArtifact(art cpi.ArtifactAccess, set cpi.ArtifactSink, tags ...string) error {
	if art.GetDescriptor().IsIndex() {
		return TransferIndex(art.IndexAccess(), set, tags...)
	} else {
		return TransferManifest(art.ManifestAccess(), set, tags...)
	}
}

func TransferIndex(art cpi.IndexAccess, set cpi.ArtifactSink, tags ...string) (err error) {
	logging.Logger().Debug("transfer OCI index", "digest", art.Digest())
	defer func() {
		logging.Logger().Debug("transfer OCI index done", "error", logging.ErrorMessage(err))
	}()

	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&err)

	for _, l := range art.GetDescriptor().Manifests {
		loop := finalize.Nested()
		logging.Logger().Debug("indexed manifest", "digest", "digest", l.Digest, "size", l.Size)
		art, err := art.GetArtifact(l.Digest)
		if err != nil {
			return errors.Wrapf(err, "getting indexed artifact %s", l.Digest)
		}
		loop.Close(art)
		err = TransferArtifact(art, set)
		if err != nil {
			return errors.Wrapf(err, "transferring indexed artifact %s", l.Digest)
		}
		err = loop.Finalize()
		if err != nil {
			return err
		}
	}
	_, err = set.AddArtifact(art, tags...)
	if err != nil {
		return errors.Wrapf(err, "transferring index artifact")
	}
	return err
}

func TransferManifest(art cpi.ManifestAccess, set cpi.ArtifactSink, tags ...string) (err error) {
	logging.Logger().Debug("transfer OCI manifest", "digest", art.Digest())
	defer func() {
		logging.Logger().Debug("transfer OCI manifest done", "error", logging.ErrorMessage(err))
	}()

	blob, err := art.GetConfigBlob()
	if err != nil {
		return errors.Wrapf(err, "getting config blob")
	}
	err = set.AddBlob(blob)
	blob.Close()
	if err != nil {
		return errors.Wrapf(err, "transferring config blob")
	}
	for i, l := range art.GetDescriptor().Layers {
		logging.Logger().Debug("layer", "digest", "digest", l.Digest, "size", l.Size, "index", i)
		blob, err = art.GetBlob(l.Digest)
		if err != nil {
			return errors.Wrapf(err, "getting layer blob %s", l.Digest)
		}
		err = set.AddBlob(blob)
		blob.Close()
		if err != nil {
			return errors.Wrapf(err, "transferring layer blob %s", l.Digest)
		}
	}
	blob, err = set.AddArtifact(art, tags...)
	if err != nil {
		return errors.Wrapf(err, "transferring image artifact")
	}
	return blob.Close()
}
