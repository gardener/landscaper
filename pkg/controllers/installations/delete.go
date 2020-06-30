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

package installations

import (
	"context"

	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

func (a *actuator) ensureDeletion(ctx context.Context, inst *installations.Installation) error {

	// check if suitable for deletion
	// - sibling has imports that we export
	// - no subinstallation

	// virtual garden
	// delete etcd, dns

	// delete execution

	// delete all subinstallations

	return nil
}
