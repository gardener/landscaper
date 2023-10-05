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
	ocmFactory    model.Factory
	cnudieFactory model.Factory

	ocmLibraryMode *bool
)

func init() {
	cnudieFactory = &cnudie.Factory{}
	ocmFactory = &ocmlib.Factory{}

	// This is for testing purposes only!
	log, _ := logging.GetLogger()
	m := os.Getenv("USE_OCM_LIB")
	if m == "true" {
		SetOCMLibraryMode(true)
		log.Info("UseOCMLib set through environment variable, this should only happen in test scenarios!", "value", true)
	} else if m == "false" {
		SetOCMLibraryMode(false)
		log.Info("UseOCMLib set through environment variable, this should only happen in test scenarios!", "value", false)
	}

	// Enable logging from ocm lib
	logging.SetLogConsumer(ocmFactory.SetApplicationLogger)
}

// SetOCMLibraryMode can only be set once as it is determined by the landscaper or deployer configuration
func SetOCMLibraryMode(useOCMLib bool) {
	log, _ := logging.GetLogger()
	if ocmLibraryMode == nil {
		ocmLibraryMode = &useOCMLib
	} else {
		log.Info("useOCMLib flag already set (can only be set once!)", "useOCMLib", *ocmLibraryMode)
	}
}

func GetFactory(useOCM ...bool) model.Factory {
	log, _ := logging.GetLogger()

	useOCMBool := utils.OptionalDefaultedBool(false, useOCM...)

	// Behavior defined as in
	// https://github.tools.sap/kubernetes/k8s-lifecycle-management/blob/master/docs/ADRs/2023-09-29-Cnudie-OCM-Switch.md
	if useOCMBool || *ocmLibraryMode {
		log.Info("using cnudie")
		return ocmFactory
	} else {
		log.Info("using ocm")
		return cnudieFactory
	}
}
