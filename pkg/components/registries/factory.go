package registries

import (
	"fmt"
	"os"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/pkg/components/ocmlib"

	"github.com/gardener/landscaper/pkg/components/cnudie"
	"github.com/gardener/landscaper/pkg/components/model"
)

const (
	LEGACY = "cnudie"
	OCM    = "ocm"
)

var (
	factory model.Factory
)

func init() {
	m := os.Getenv("LANDSCAPER_LIBRARY_MODE")
	if m == "" {
		m = LEGACY
	}
	if err := SetFactory(m); err != nil {
		panic(fmt.Sprintf("LANDSCAPER_LIBRARY_MODE: %s", m))
	}
	logging.SetLogConsumer((&ocmlib.Factory{}).SetApplicationLogger)
}

func SetFactory(mode string) error {
	switch mode {
	case LEGACY:
		factory = &cnudie.Factory{}
	case OCM:
		factory = &ocmlib.Factory{}
	default:
		return fmt.Errorf("invalid factory LANDSCAPER_LIBRARY_MODE %q", mode)
	}
	return nil
}

func GetFactory() model.Factory {
	return factory
}
