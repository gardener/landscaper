// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"runtime"
	"strings"

	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

var (
	gitVersion   = "0.0.0-dev"
	gitCommit    string
	gitTreeState string
	buildDate    = "1970-01-01T00:00:00Z"
)

// GetInterface returns the overall codebase version. It's for detecting
// what code a binary was built from.
// These variables typically come from -ldflags settings and in
// their absence fallback to the settings in pkg/version/base.go
func Get() apimachineryversion.Info {
	var (
		version  = strings.Split(gitVersion, ".")
		gitMajor string
		gitMinor string
	)

	if len(version) >= 2 {
		gitMajor = version[0]
		gitMinor = version[1]
	}

	return apimachineryversion.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
