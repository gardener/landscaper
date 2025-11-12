// Copyright 2022 Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package signatures

import (
	"context"
	"fmt"
	"hash"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

// Signer interface is used to implement different signing algorithms.
// Each Signer should have a matching Verifier.
type Signer interface {
	// Sign returns the signature for the data for the component-descriptor
	Sign(componentDescriptor cdv2.ComponentDescriptor, digest cdv2.DigestSpec) (*cdv2.SignatureSpec, error)
}

// Verifier interface is used to implement different verification algorithms.
// Each Verifier should have a matching Signer.
type Verifier interface {
	// Verify checks the signature, returns an error on verification failure
	Verify(componentDescriptor cdv2.ComponentDescriptor, signature cdv2.Signature) error
}

// Hasher encapsulates a hash.Hash interface with an algorithm name.
type Hasher struct {
	HashFunction  hash.Hash
	AlgorithmName string
}

// HasherForName creates a Hasher instance for the algorithmName.
func HasherForName(algorithmName string) (*Hasher, error) {
	hashfunc, ok := HashFunctions[algorithmName]
	if !ok {
		return nil, fmt.Errorf("hash algorithm %s not found/implemented", algorithmName)
	}

	return &Hasher{
		HashFunction:  hashfunc.New(),
		AlgorithmName: algorithmName,
	}, nil
}

type ResourceDigester interface {
	DigestForResource(ctx context.Context, componentDescriptor cdv2.ComponentDescriptor, resource cdv2.Resource, hasher Hasher) (*cdv2.DigestSpec, error)
}
