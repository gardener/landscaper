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

package helm

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// Type is the type name of the deployer.
const Type lsv1alpha1.ExecutionType = "Mock"

// ProviderConfiguration is the configuration of a helm deploy item.
// todo: use versioned configuration
type Configuration struct {
	// Phase sets the phase of the DeployItem
	Phase *lsv1alpha1.ExecutionPhase `json:"phase,omitempty"`

	// ProviderStatus sets the provider status to the given value
	ProviderStatus *runtime.RawExtension `json:"providerStatus,omitempty"`

	// Export sets the exported configuration to the given value
	Export *json.RawMessage `json:"export,omitempty"`
}
