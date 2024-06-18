// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"os"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/components/ocmlib"

	"github.com/gardener/landscaper/pkg/components/model"
)

var (
	ocmFactory     model.Factory = &ocmlib.Factory{}
	ocmLibraryMode *bool
)

func init() {
	// This is for testing purposes only!
	m := os.Getenv("USE_OCM_LIB")
	if m == "true" {
		SetOCMLibraryMode(true)
	} else if m == "false" {
		SetOCMLibraryMode(false)
	}
}

// SetOCMLibraryMode can only be set once as it is determined by the landscaper or deployer configuration
func SetOCMLibraryMode(useOCMLib bool) {
	if ocmLibraryMode == nil {
		ocmLibraryMode = &useOCMLib
	}
}

func GetFactory(useOCM ...bool) model.Factory {
	log, _ := logging.GetLogger()

	log.Info("using ocm")
	return ocmFactory
}
