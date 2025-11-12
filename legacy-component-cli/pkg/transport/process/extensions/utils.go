// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package extensions

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
)

const (
	// ExecutableType defines the type of an executable
	ExecutableType = "Executable"
)

// CreateExecutable creates a new executable defined by a spec
func CreateExecutable(rawSpec *json.RawMessage) (process.ResourceStreamProcessor, error) {
	type executableSpec struct {
		Bin  string
		Args []string
		Env  map[string]string
	}

	var spec executableSpec
	if err := yaml.Unmarshal(*rawSpec, &spec); err != nil {
		return nil, fmt.Errorf("unable to parse spec: %w", err)
	}

	return NewUnixDomainSocketExecutable(spec.Bin, spec.Args, spec.Env)
}
