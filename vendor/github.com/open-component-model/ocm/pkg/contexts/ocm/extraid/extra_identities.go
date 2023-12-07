// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package extraid

import (
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
)

const (
	SystemIdentityName    = metav1.SystemIdentityName
	SystemIdentityVersion = metav1.SystemIdentityVersion

	// ExecutableOperatingSystem is the standard extra id for the operating system for an executable.
	ExecutableOperatingSystem = "os"
	// ExecutableArchitecture is the standard extra id for the architecture system for an executable.
	ExecutableArchitecture = "architecture"
)
