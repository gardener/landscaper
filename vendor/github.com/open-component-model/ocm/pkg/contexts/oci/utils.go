// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"

	ociidentity "github.com/open-component-model/ocm/pkg/contexts/credentials/builtin/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
	"github.com/open-component-model/ocm/pkg/runtime"
)

func AsTags(tag string) []string {
	if tag != "" {
		return []string{tag}
	}
	return nil
}

func StandardOCIRef(host, repository, version string) string {
	sep := grammar.TagSeparator
	if ok, _ := artdesc.IsDigest(version); ok {
		sep = grammar.DigestSeparator
	}
	return fmt.Sprintf("%s%s%s%s%s", host, grammar.RepositorySeparator, repository, sep, version)
}

func IsIntermediate(spec RepositorySpec) bool {
	if s, ok := spec.(IntermediateRepositorySpecAspect); ok {
		return s.IsIntermediate()
	}
	return false
}

func IsUnknown(r RepositorySpec) bool {
	return runtime.IsUnknown(r)
}

func GetConsumerIdForRef(ref string) (cpi.ConsumerIdentity, error) {
	r, err := ParseRef(ref)
	if err != nil {
		return nil, err
	}
	return ociidentity.GetConsumerId(r.Host, r.Repository), nil
}
