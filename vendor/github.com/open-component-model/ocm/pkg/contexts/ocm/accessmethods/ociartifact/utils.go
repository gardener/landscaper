// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociartifact

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/oci/grammar"
)

func Hint(nv common.NameVersion, locator, repo, version string) string {
	repository := fmt.Sprintf("%s/%s", nv.GetName(), locator)
	if repo != "" {
		if strings.HasPrefix(repo, grammar.RepositorySeparator) {
			repository = repo[1:]
		} else {
			repository = fmt.Sprintf("%s/%s", nv.GetName(), repo)
		}
	}
	if !strings.Contains(repository, ":") {
		repository = fmt.Sprintf("%s:%s", repository, version)
	}
	return repository
}
