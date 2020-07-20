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
	"bytes"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"sort"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

type generation struct {
	// Kubernetes generation of the respective Installation resource (`.metadata.generation`).
	// Used to detect any changes in the installation's spec.
	Generation int64

	// Imports are all states of imports defined in the the Installations DefintionsRef.
	// The array must be ordered by its key.
	Imports imports
}

type importState struct {
	// Key is the import key of the ComponentDefinition
	Key string

	// Generation is the config generation of the installation where the import's coming from.
	// The hash of the static data is used if the import is coming from static data.
	Generation string
}

type imports []importState

var _ sort.Interface = imports{}

func (i imports) Len() int { return len(i) }

func (i imports) Swap(a, b int) { i[a], i[b] = i[b], i[a] }

func (i imports) Less(a, b int) bool {
	return i[a].Key < i[b].Key
}

// CreateGenerationHash creates a unique generation for a Installation.
// That has is based in the Installation's Spec and its import state.
func CreateGenerationHash(inst *lsv1alpha1.Installation) (string, error) {
	gen := generation{
		Generation: inst.GetGeneration(),
		Imports:    make([]importState, len(inst.Status.Imports)),
	}

	for i, state := range inst.Status.Imports {
		gen.Imports[i] = importState{
			Key:        state.To,
			Generation: state.ConfigGeneration,
		}
	}
	sort.Sort(gen.Imports)

	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(gen); err != nil {
		return "", err
	}

	h := sha1.New()
	h.Write(data.Bytes())
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
