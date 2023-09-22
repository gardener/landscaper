package registries

import (
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/components/ocmlib"

	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
)

var (
	factory        model.Factory
	ocmLibraryMode bool
)

func init() {
	SetOCMLibraryMode(false)
	logging.SetLogConsumer((&ocmlib.Factory{}).SetApplicationLogger)
}

func SetOCMLibraryMode(useOCMLib bool) {
	ocmLibraryMode = useOCMLib
}

func SetFactory(useOCM bool) {
	if useOCM || ocmLibraryMode {
		factory = &ocmlib.Factory{}
	}
	factory = &cnudie.Factory{}
}

func GetFactory() model.Factory {
	return factory
}
