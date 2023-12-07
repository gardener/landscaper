// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cpi

import (
	"io"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/iotools"
)

type AccessMethodSource interface {
	AccessMethod() (AccessMethod, error)
}

// ResourceReader gets a Reader for a given resource/source access.
// It provides a Reader handling the Close contract for the access method
// by connecting the access method's Close method to the Readers Close method .
// Deprecated: use GetResourceReader.
// It must be deprecated because of the support of free-floating ReSourceAccess
// implementations, they not necessarily provide an AccessMethod.
func ResourceReader(s AccessMethodSource) (io.ReadCloser, error) {
	meth, err := s.AccessMethod()
	if err != nil {
		return nil, err
	}
	return toResourceReaderForMethod(meth)
}

// ResourceMimeReader gets a Reader for a given resource/source access.
// It provides a Reader handling the Close contract for the access method
// by connecting the access method's Close method to the Readers Close method.
// Additionally, the mime type is returned.
// Deprecated: use GetResourceMimeReader.
// It must be deprecated because of the support of free-floating ReSourceAccess
// implementations, they not necessarily provide an AccessMethod.
func ResourceMimeReader(s AccessMethodSource) (io.ReadCloser, string, error) {
	meth, err := s.AccessMethod()
	if err != nil {
		return nil, "", err
	}
	r, err := toResourceReaderForMethod(meth)
	return r, meth.MimeType(), err
}

func toResourceReaderForMethod(meth AccessMethod) (io.ReadCloser, error) {
	r, err := meth.Reader()
	if err != nil {
		meth.Close()
		return nil, err
	}
	return iotools.AddReaderCloser(r, meth, "access method"), nil
}

// GetResourceMimeReader gets a Reader for a given resource/source access.
// It provides a Reader handling the Close contract for the access method.
func GetResourceReader(acc AccessProvider) (io.ReadCloser, error) {
	return blobaccess.ReaderFromProvider(acc)
}

// GetResourceMimeReader gets a Reader for a given resource/source access.
// It provides a Reader handling the Close contract for the access method.
// Additionally, the mime type is returned.
func GetResourceMimeReader(acc AccessProvider) (io.ReadCloser, string, error) {
	return blobaccess.MimeReaderFromProvider(acc)
}

////////////////////////////////////////////////////////////////////////////////

func ArtifactNameHint(spec AccessSpec, cv ComponentVersionAccess) string {
	if h, ok := spec.(HintProvider); ok {
		return h.GetReferenceHint(cv)
	}
	return ""
}

func ReferenceHint(spec AccessSpec, cv ComponentVersionAccess) string {
	if h, ok := spec.(internal.HintProvider); ok {
		return h.GetReferenceHint(cv)
	}
	return ""
}

func GlobalAccess(spec AccessSpec, ctx Context) AccessSpec {
	g := spec.GlobalAccessSpec(ctx)
	if g != nil && g.IsLocal(ctx) {
		g = nil
	}
	return g
}
