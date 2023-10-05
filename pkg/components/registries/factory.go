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

const (
	OCM    = "true"
	CNUDIE = "false"
)

var (
	ocmFactory     *ocmlib.Factory
	cnudieFactory  *cnudie.Factory
	defaultFactory model.Factory

	ocmLibraryMode *bool
)

func init() {
	m := os.Getenv("USE_OCM_LIB")
	if m == CNUDIE || m == "" {
		defaultFactory = &cnudie.Factory{}
	} else if m == OCM {
		defaultFactory = &ocmlib.Factory{}
	} else {
		defaultFactory = nil
	}

	cnudieFactory = &cnudie.Factory{}
	ocmFactory = &ocmlib.Factory{}

	// Enable logging from ocm lib
	logging.SetLogConsumer(ocmFactory.SetApplicationLogger)
}

// SetOCMLibraryMode can only be set once as it is determined by the landscaper or deployer configuration
func SetOCMLibraryMode(useOCMLib bool) {
	log, _ := logging.GetLogger()
	if ocmLibraryMode == nil {
		*ocmLibraryMode = useOCMLib
	} else {
		log.Info("useOCMLib flag already set to ", "useOCMLib", *ocmLibraryMode)
	}
}

func GetFactory(useOCM ...bool) model.Factory {
	log, _ := logging.GetLogger()
	// This is to centrally control the facade implementation used by unit tests.
	// It assumes that if ocmLibraryMode is not set and therefore nil, GetFactory is
	// not called in an actually running landscaper instance.
	// If no argument is passed, useOCM is defaulted to false and
	// the library defined by the environment variable will be used.
	useOCMBool := utils.OptionalDefaultedBool(false, useOCM...)
	if ocmLibraryMode == nil {
		log.Info("useOCMLib flag not set, this should only happen in tests")
		if useOCMBool {
			log.Info("using ocm-lib")
			return ocmFactory
		} else {
			log.Info("using cnudie")
			return defaultFactory
		}
	}

	// Behavior defined as in
	// https://github.tools.sap/kubernetes/k8s-lifecycle-management/blob/master/docs/ADRs/2023-09-29-Cnudie-OCM-Switch.md
	if useOCMBool || *ocmLibraryMode {
		log.Info("using cnudie")
		return cnudieFactory
	} else {
		log.Info("using ocm")
		return ocmFactory
	}
}
