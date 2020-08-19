// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

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
