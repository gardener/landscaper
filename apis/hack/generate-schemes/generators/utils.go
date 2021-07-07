// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package generators

import (
	"fmt"
	"strings"
)

type PackageVersionName struct {
	Name string
	Version string
	Package string
}

// ParsePackageVersionName parses the name from the openapi definition.
// name is expected to be some/path/<package>/<version>.<name>
func ParsePackageVersionName(name string) PackageVersionName {
	splitName := strings.Split(name, "/")
	if len(splitName) < 2 {
		panic(fmt.Errorf("a component name must consits of at least a package identifier and a name.version but got %s", name))
	}
	versionName := splitName[len(splitName) - 1]
	packageName := splitName[len(splitName) - 2]

	versionNameSplit := strings.Split(versionName, ".")
	if len(versionNameSplit) != 2 {
		panic(fmt.Errorf("a component name must consits of name.version but got %s", versionName))
	}
	return PackageVersionName{
		Package: packageName,
		Name: versionNameSplit[1],
		Version: versionNameSplit[0],
	}
}

// String implements the stringer method.
func (pvn PackageVersionName) String() string {
	return fmt.Sprintf("%s-%s-%s", pvn.Package, pvn.Version, pvn.Name)
}
