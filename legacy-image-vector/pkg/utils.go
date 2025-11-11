// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
)

// TagIsDigest validates of a tag is a valid digest.
func TagIsDigest(tag string) bool {
	_, err := digest.Parse(tag)
	return err == nil
}

// ParseImageRef parses a valid image ref into its repository and version
func ParseImageRef(ref string) (repository, version string, err error) {
	// check if the ref contains a digest
	if strings.Contains(ref, "@") {
		splitRef := strings.Split(ref, "@")
		if len(splitRef) != 2 {
			return "", "", fmt.Errorf("invalid image reference %q, expected only 1 char of '@'", ref)
		}
		return splitRef[0], splitRef[1], nil
	}
	splitRef := strings.Split(ref, ":")
	if len(splitRef) > 3 {
		return "", "", fmt.Errorf("invalid image reference %q, expected maximum 3 chars of ':'", ref)
	}

	repository = strings.Join(splitRef[:(len(splitRef)-1)], ":")
	version = splitRef[len(splitRef)-1]
	err = nil
	return
}
