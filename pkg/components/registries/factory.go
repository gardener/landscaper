// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"os"

	"github.com/open-component-model/ocm/pkg/utils"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/components/ocmlib"

	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
)

var (
	ocmFactory    model.Factory = &ocmlib.Factory{}
	cnudieFactory model.Factory = &cnudie.Factory{}

	ocmLibraryMode *bool
)

func init() {
	// Enable logging with our logger from ocm lib
	// The config of the logger cannot be accessed in the init function, as the logger might not have been
	// initialized itself yet. Therefore, we pass a function through which the logger can be passed to the ocmlib
	// after it is fully configured.
	logging.SetLogConsumer(ocmFactory.SetApplicationLogger)

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

	var ocmLibraryModeBool bool
	if ocmLibraryMode != nil {
		ocmLibraryModeBool = *ocmLibraryMode
	} else {
		ocmLibraryModeBool = false
		log.Info("useOCMLib flag not set and therefore defaulted!", "ocmLibraryMode", ocmLibraryModeBool)
	}
	useOCMBool := utils.OptionalDefaultedBool(false, useOCM...)

	if useOCMBool || ocmLibraryModeBool {
		log.Info("using cnudie")
		return ocmFactory
	} else {
		log.Info("using ocm")
		return cnudieFactory
	}
}
