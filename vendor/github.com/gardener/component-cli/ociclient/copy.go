// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient

import (
	"context"
	"fmt"
	"io"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Copy copies a oci artifact from one location to a target ref.
// The artifact is copied without any modification.
// This function does directly stream the blobs from the upstream it does not use any cache.
func Copy(ctx context.Context, client Client, from, to string) error {
	artifact, err := client.GetOCIArtifact(ctx, from)
	if err != nil {
		return fmt.Errorf("unable to get source oci artifact %q: %w", from, err)
	}

	store := GenericStore(func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error {
		return client.Fetch(ctx, from, desc, writer)
	})

	return client.PushOCIArtifact(ctx, to, artifact, WithStore(store))
}

// GenericStore is a helper struct to implement a custom oci blob store.
type GenericStore func(ctx context.Context, desc ocispecv1.Descriptor, writer io.Writer) error

func (s GenericStore) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	ctx := context.Background()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		defer ctx.Done()
		if err := s(ctx, desc, writer); err != nil {
			_ = reader.CloseWithError(err)
		}
	}()
	return reader, nil
}
