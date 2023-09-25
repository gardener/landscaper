// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registries

import (
	"context"
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
	SetOCMLibraryMode(true)
	SetFactory(context.Background(), false)
	log, _ := logging.GetLogger()
	if _, ok := factory.(*ocmlib.Factory); ok {
		log.Info("set ocmlib during initialization")
	} else {
		log.Info("set cnudie during initialization")
	}
	logging.SetLogConsumer((&ocmlib.Factory{}).SetApplicationLogger)
}

func SetOCMLibraryMode(useOCMLib bool) {
	ocmLibraryMode = useOCMLib
}

func SetFactory(ctx context.Context, useOCM bool) {
	log := logging.FromContextOrDiscard(ctx)
	if useOCM || ocmLibraryMode {
		factory = &ocmlib.Factory{}
		log.Info("using ocmlib")
	} else {
		factory = &cnudie.Factory{}
		log.Info("using cnudie")
	}
}

func GetFactory() model.Factory {
	return factory
}
