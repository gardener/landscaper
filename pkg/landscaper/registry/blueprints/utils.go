// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry

import (
	"errors"
	"strings"
)

// VersionedName represents a reference to a ComponentDefinition
type VersionedName struct {
	Name    string
	Version string
}

// ParseDefinitionRef parses a Blueprint reference of the form "name:version"
func ParseDefinitionRef(ref string) (VersionedName, error) {
	splitName := strings.Split(ref, ":")

	if len(splitName) != 2 && len(splitName) != 3 {
		return VersionedName{}, NewVersionParseError(ref, errors.New("invalid ref format"))
	}

	return VersionedName{
		Name:    strings.Join(splitName[:len(splitName)-1], ":"),
		Version: splitName[len(splitName)-1],
	}, nil
}
