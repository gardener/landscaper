// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package uploaders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	processutils "github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/utils"
)

type localOCIBlobUploader struct {
	client    ociclient.Client
	targetCtx cdv2.OCIRegistryRepository
}

func NewLocalOCIBlobUploader(client ociclient.Client, targetCtx cdv2.OCIRegistryRepository) (process.ResourceStreamProcessor, error) {
	if client == nil {
		return nil, errors.New("client must not be nil")
	}

	obj := localOCIBlobUploader{
		targetCtx: targetCtx,
		client:    client,
	}
	return &obj, nil
}

func (d *localOCIBlobUploader) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, blobreader, err := processutils.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read processor message: %w", err)
	}
	if blobreader == nil {
		return errors.New("resource blob must not be nil")
	}
	defer blobreader.Close()

	tmpfile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("unable to create tempfile: %w", err)
	}
	defer tmpfile.Close()

	size, err := io.Copy(tmpfile, blobreader)
	if err != nil {
		return fmt.Errorf("unable to copy resource blob to tempfile: %w", err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	dgst, err := digest.FromReader(tmpfile)
	if err != nil {
		return fmt.Errorf("unable to calculate digest: %w", err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	desc := ocispecv1.Descriptor{
		Digest:    dgst,
		Size:      int64(size),
		MediaType: res.Type,
	}

	if err := d.uploadLocalOCIBlob(ctx, cd, res, tmpfile, desc); err != nil {
		return fmt.Errorf("unable to upload blob: %w", err)
	}

	acc, err := cdv2.NewUnstructured(cdv2.NewLocalOCIBlobAccess(dgst.String()))
	if err != nil {
		return fmt.Errorf("unable to create resource access object: %w", err)
	}
	res.Access = &acc

	if _, err := tmpfile.Seek(0, 0); err != nil {
		return fmt.Errorf("unable to seek to beginning of tempfile: %w", err)
	}

	if err := processutils.WriteProcessorMessage(*cd, res, tmpfile, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}

func (d *localOCIBlobUploader) uploadLocalOCIBlob(ctx context.Context, cd *cdv2.ComponentDescriptor, res cdv2.Resource, r io.Reader, desc ocispecv1.Descriptor) error {
	targetRef := utils.CalculateBlobUploadRef(d.targetCtx, cd.Name, cd.Version)

	store := ociclient.GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		_, err := io.Copy(writer, r)
		return err
	})

	if err := d.client.PushBlob(ctx, targetRef, desc, ociclient.WithStore(store)); err != nil {
		return fmt.Errorf("unable to push blob: %w", err)
	}

	return nil
}
