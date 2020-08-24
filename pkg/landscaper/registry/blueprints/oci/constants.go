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

package oci

import "fmt"

const (
	// ComponentDefinitionConfigMediaType is the reserved media type for the ComponentDefinition manifest config
	ComponentDefinitionConfigMediaType = "application/vnd.gardener.landscaper.componentdefinition.v1+json"

	// ComponentDefinitionAnnotationTitleContent is the name of the annotation title of the layer that contains the content blob.
	ComponentDefinitionAnnotationTitleContent = "content"
)

// ref creates a reference from a name and a version
func makeRef(name, version string) string {
	return fmt.Sprintf("%s:%s", name, version)
}
