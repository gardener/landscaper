// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imports

import (
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/landscapeconfig"
)

// Validators is a struct that contains everything to
// validate if all imports of a installation are satisfied.
type Validator struct {
	*installations.Operation

	lsConfig *landscapeconfig.LandscapeConfig
	parent   *installations.Installation
	siblings []*installations.Installation
}

// Constructor is a struct that contains all values
// that are needed to load all imported data and
// generate the one imported config
type Constructor struct {
	*installations.Operation
	validator *Validator

	lsConfig *landscapeconfig.LandscapeConfig
	parent   *installations.Installation
	siblings []*installations.Installation
}
