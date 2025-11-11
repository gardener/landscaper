// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package filters

import (
	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

// Filter defines the interface for matching component resources with downloaders, processing rules, and uploaders
type Filter interface {
	// Matches matches a component descriptor and a resource against the filter
	Matches(cdv2.ComponentDescriptor, cdv2.Resource) bool
}
