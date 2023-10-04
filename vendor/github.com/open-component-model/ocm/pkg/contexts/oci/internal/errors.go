// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	KIND_OCIARTIFACT = "oci artifact"
	KIND_BLOB        = accessio.KIND_BLOB
	KIND_MEDIATYPE   = accessio.KIND_MEDIATYPE
)

func ErrUnknownArtifact(name, version string) error {
	return errors.ErrUnknown(KIND_OCIARTIFACT, fmt.Sprintf("%s:%s", name, version))
}
