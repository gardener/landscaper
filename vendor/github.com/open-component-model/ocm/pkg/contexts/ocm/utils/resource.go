// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

func GetResourceData(acc ocm.ResourceAccess) ([]byte, error) {
	m, err := acc.AccessMethod()
	if err != nil {
		return nil, err
	}
	defer m.Close()
	return m.Get()
}
