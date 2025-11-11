// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"fmt"
	"os"
	"path/filepath"
)

// CliHomeEnvName is the name of the environment variable that configures the component cli home directory.
const CliHomeEnvName = "COMPONENT_CLI_HOME"

// ComponentDescriptorPathEnvName is the name of the environment variable that contains the absolute file path to output the final descriptor to.
const ComponentDescriptorPathEnvName = "COMPONENT_DESCRIPTOR_PATH"

// BaseDefinitionPathEnvName is the name of the environment variable that contains the absolute file path to the base component descriptor
const BaseDefinitionPathEnvName = "BASE_DEFINITION_PATH"

// ComponentArchivePathEnvName is the name of the environment variable that contains the file path to the component archive to be used.
const ComponentArchivePathEnvName = "COMPONENT_ARCHIVE_PATH"

// ComponentRepositoryRepositoryBaseUrlEnvName is the name of the environment variable that hold the base url of the current component repository
const ComponentRepositoryRepositoryBaseUrlEnvName = "COMPONENT_REPOSITORY_BASE_URL"

// ComponentRepositoryCacheDirEnvVar is the name of the environment variable that points to the local component descriptor cache.
const ComponentRepositoryCacheDirEnvVar = "COMPONENT_REPOSITORY_CACHE_DIR"

// CliHomeDir returns the home directoy of the components cli.
// It returns the COMPONENT_CLI_HOME if its defined otherwise
// the default "$HOME/.component-cli" is returned.
func CliHomeDir() (string, error) {
	lsHome := os.Getenv(CliHomeEnvName)
	if len(lsHome) != 0 {
		return lsHome, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine the component home directory: %w", err)
	}
	return filepath.Join(homeDir, ".component-cli"), nil
}
