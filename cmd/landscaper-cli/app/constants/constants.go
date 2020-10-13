// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"fmt"
	"os"
	"path/filepath"
)

// LandscaperCliHomeEnvName is the name of the environment variable that configures the landscaper cli home directory.
const LandscaperCliHomeEnvName = "LANDSCAPER_HOME"

// LandscaperCliHomeDir returns the home directoy of the landscpaer cli.
// It returns the $LANDSCAPER_HOME if its defined otherwise
// the default "$HOME/.landscaper" is returned.
func LandscaperCliHomeDir() (string, error) {
	lsHome := os.Getenv(LandscaperCliHomeEnvName)
	if len(lsHome) != 0 {
		return lsHome, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine the landscaper home directory: %w", err)
	}
	return filepath.Join(homeDir, ".landscaper"), nil
}
