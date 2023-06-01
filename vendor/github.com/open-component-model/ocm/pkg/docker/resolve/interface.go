// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resolve

import (
	"context"
	"io"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// all new and modified

type Source interface {
	Reader() (io.ReadCloser, error)
}

// Resolver provides remotes based on a locator.
type Resolver interface {
	// Resolve attempts to resolve the reference into a name and descriptor.
	//
	// The argument `ref` should be a scheme-less URI representing the remote.
	// Structurally, it has a host and path. The "host" can be used to directly
	// reference a specific host or be matched against a specific handler.
	//
	// The returned name should be used to identify the referenced entity.
	// Dependending on the remote namespace, this may be immutable or mutable.
	// While the name may differ from ref, it should itself be a valid ref.
	//
	// If the resolution fails, an error will be returned.
	Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error)

	// Fetcher returns a new fetcher for the provided reference.
	// All content fetched from the returned fetcher will be
	// from the namespace referred to by ref.
	Fetcher(ctx context.Context, ref string) (Fetcher, error)

	// Pusher returns a new pusher for the provided reference
	// The returned Pusher should satisfy content.Ingester and concurrent attempts
	// to push the same blob using the Ingester API should result in ErrUnavailable.
	Pusher(ctx context.Context, ref string) (Pusher, error)

	Lister(ctx context.Context, ref string) (Lister, error)
}

// Fetcher fetches content.
type Fetcher interface {
	// Fetch the resource identified by the descriptor.
	Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error)
}

// Pusher pushes content
// don't use write interface of containerd remotes.Pusher.
type Pusher interface {
	// Push returns a push request for the given resource identified
	// by the descriptor and the given data source.
	Push(ctx context.Context, d ocispec.Descriptor, src Source) (PushRequest, error)
}

type Lister interface {
	List(context.Context) ([]string, error)
}

// PushRequest handles the result of a push request
// replaces containerd content.Writer.
type PushRequest interface {
	// Commit commits the blob (but no roll-back is guaranteed on an error).
	// size and expected can be zero-value when unknown.
	// Commit always closes the writer, even on error.
	// ErrAlreadyExists aborts the writer.
	Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error

	// Status returns the current state of write
	Status() (content.Status, error)
}
