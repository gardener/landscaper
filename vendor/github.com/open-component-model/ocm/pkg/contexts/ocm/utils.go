// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"io"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/internal"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

////////////////////////////////////////////////////////////////////////////////

func AssureTargetRepository(session Session, ctx Context, targetref string, opts ...interface{}) (Repository, error) {
	var format accessio.FileFormat
	var archive string
	var fs vfs.FileSystem

	for _, o := range opts {
		switch v := o.(type) {
		case vfs.FileSystem:
			if fs == nil && v != nil {
				fs = v
			}
		case accessio.FileFormat:
			format = v
		case string:
			archive = v
		default:
			return nil, fmt.Errorf("invalid option type %T", o)
		}
	}

	ref, err := ParseRepo(targetref)
	if err != nil {
		return nil, err
	}
	if ref.TypeHint == "" {
		ref.TypeHint = archive
	}
	if format != "" && ref.TypeHint != "" && !strings.Contains(ref.TypeHint, "+") {
		for _, f := range ctf.SupportedFormats() {
			if f == format {
				ref.TypeHint += "+" + format.String()
			}
		}
	}
	ref.CreateIfMissing = true
	target, err := session.DetermineRepositoryBySpec(ctx, &ref)
	if err != nil {
		if !errors.IsErrUnknown(err) || vfs.IsErrNotExist(err) || ref.Info == "" {
			return nil, err
		}
		if ref.Type == "" {
			ref.Type = format.String()
		}
		if ref.Type == "" {
			return nil, fmt.Errorf("ctf format type required to create ctf")
		}
		target, err = ctf.Create(ctx, accessobj.ACC_CREATE, ref.Info, 0o770, accessio.PathFileSystem(utils.FileSystem(fs)))
		if err != nil {
			return nil, err
		}
		session.Closer(target)
	}
	return target, nil
}

type AccessMethodSource = cpi.AccessMethodSource

// ResourceReader gets a Reader for a given resource/source access.
// It provides a Reader handling the Close contract for the access method
// by connecting the access method's Close method to the Readers Close method .
// Deprecated: use ocmutils.GetResourceReader.
// It must be deprecated because of the support of free-floating ReSourceAccess
// implementations, they not necessarily provide an AccessMethod.
func ResourceReader(s AccessMethodSource) (io.ReadCloser, error) {
	return cpi.ResourceReader(s)
}

func IsIntermediate(spec RepositorySpec) bool {
	if s, ok := spec.(IntermediateRepositorySpecAspect); ok {
		return s.IsIntermediate()
	}
	return false
}

func ComponentRefKey(ref *compdesc.ComponentReference) common.NameVersion {
	return common.NewNameVersion(ref.GetComponentName(), ref.GetVersion())
}

func IsUnknownRepositorySpec(s RepositorySpec) bool {
	return runtime.IsUnknown(s)
}

func IsUnknownAccessSpec(s AccessSpec) bool {
	return runtime.IsUnknown(s)
}

func WrapContextProvider(ctx LocalContextProvider) ContextProvider {
	return internal.WrapContextProvider(ctx)
}

func ReferenceHint(spec AccessSpec, cv ComponentVersionAccess) string {
	if h, ok := spec.(internal.HintProvider); ok {
		return h.GetReferenceHint(cv)
	}
	return ""
}

func GlobalAccess(spec AccessSpec, ctx Context) AccessSpec {
	if h, ok := spec.(internal.GlobalAccessProvider); ok {
		return h.GlobalAccessSpec(ctx)
	}
	return nil
}
