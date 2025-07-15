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
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	cdv2 "github.com/gardener/landscaper/component-spec-bindings-go/apis/v2"
)

// Entry is used for normalisation and has to contain one key
type Entry map[string]interface{}

// AddDigestsToComponentDescriptor adds digest to componentReferences and resources as returned in the resolver functions. If a digest already exists, a mismatch against the resolved digest will return an error.
func AddDigestsToComponentDescriptor(ctx context.Context, cd *cdv2.ComponentDescriptor,
	compRefResolver func(context.Context, cdv2.ComponentDescriptor, cdv2.ComponentReference) (*cdv2.DigestSpec, error),
	resResolver func(context.Context, cdv2.ComponentDescriptor, cdv2.Resource) (*cdv2.DigestSpec, error)) error {

	for i, reference := range cd.ComponentReferences {
		digest, err := compRefResolver(ctx, *cd, reference)
		if err != nil {
			return fmt.Errorf("unable to resolve component reference for %s:%s: %w", reference.Name, reference.Version, err)
		}
		if reference.Digest != nil && !reflect.DeepEqual(reference.Digest, digest) {
			return fmt.Errorf("calculated digest mismatches existing digest for component reference %s:%s", reference.ComponentName, reference.Version)
		}
		cd.ComponentReferences[i].Digest = digest
	}

	for i, res := range cd.Resources {
		// special digest notation indicates to not digest the content
		if res.Digest != nil && reflect.DeepEqual(res.Digest, cdv2.NewExcludeFromSignatureDigest()) {
			continue
		}

		digest, err := resResolver(ctx, *cd, res)
		if err != nil {
			return fmt.Errorf("unable to resolve resource %s:%s: %w", res.Name, res.Version, err)
		}
		if res.Digest != nil && !reflect.DeepEqual(res.Digest, digest) {
			return fmt.Errorf("calculated digest mismatches existing digest for resource %s:%s", res.Name, res.Version)
		}
		cd.Resources[i].Digest = digest
	}
	return nil
}

// HashForComponentDescriptor return the hash for the component-descriptor, if it is normaliseable
// (= componentReferences and resources contain digest field)
func HashForComponentDescriptor(cd cdv2.ComponentDescriptor, hash Hasher) (*cdv2.DigestSpec, error) {
	normalisedComponentDescriptor, err := normaliseComponentDescriptor(cd)
	if err != nil {
		return nil, fmt.Errorf("unable to normalise component descriptor: %w", err)
	}
	hash.HashFunction.Reset()
	if _, err = hash.HashFunction.Write(normalisedComponentDescriptor); err != nil {
		return nil, fmt.Errorf("unable to hash normalised component descriptor: %w", err)
	}
	return &cdv2.DigestSpec{
		HashAlgorithm:          hash.AlgorithmName,
		NormalisationAlgorithm: string(cdv2.JsonNormalisationV1),
		Value:                  hex.EncodeToString(hash.HashFunction.Sum(nil)),
	}, nil
}

func normaliseComponentDescriptor(cd cdv2.ComponentDescriptor) ([]byte, error) {
	if err := isNormaliseable(cd); err != nil {
		return nil, fmt.Errorf("component descriptor %s:%s is not normaliseable: %w", cd.Name, cd.Version, err)
	}

	meta := []Entry{
		{"schemaVersion": cd.Metadata.Version},
	}

	componentReferences := []interface{}{}
	for _, ref := range cd.ComponentSpec.ComponentReferences {
		extraIdentity := buildExtraIdentity(ref.ExtraIdentity)

		digest := []Entry{
			{"hashAlgorithm": ref.Digest.HashAlgorithm},
			{"normalisationAlgorithm": ref.Digest.NormalisationAlgorithm},
			{"value": ref.Digest.Value},
		}

		componentReference := []Entry{
			{"componentName": ref.ComponentName},
			{"name": ref.Name},
			{"version": ref.Version},
			{"extraIdentity": extraIdentity},
			{"digest": digest},
		}
		componentReferences = append(componentReferences, componentReference)
	}

	resources := []interface{}{}
	for _, res := range cd.ComponentSpec.Resources {
		extraIdentity := buildExtraIdentity(res.ExtraIdentity)

		//ignore access.type=None for normalisation and hash calculation
		if res.Access == nil || res.Access.Type == "None" {
			resource := []Entry{
				{"name": res.Name},
				{"version": res.Version},
				{"type": res.Type},
				{"relation": res.Relation},
				{"extraIdentity": extraIdentity},
			}
			resources = append(resources, resource)
			continue
		}

		digest := []Entry{
			{"hashAlgorithm": res.Digest.HashAlgorithm},
			{"normalisationAlgorithm": res.Digest.NormalisationAlgorithm},
			{"value": res.Digest.Value},
		}

		resource := []Entry{
			{"name": res.Name},
			{"version": res.Version},
			{"type": res.Type},
			{"relation": res.Relation},
			{"extraIdentity": extraIdentity},
			{"digest": digest},
		}
		resources = append(resources, resource)
	}

	componentSpec := []Entry{
		{"name": cd.ComponentSpec.Name},
		{"version": cd.ComponentSpec.Version},
		{"provider": cd.ComponentSpec.Provider},
		{"componentReferences": componentReferences},
		{"resources": resources},
	}

	normalisedComponentDescriptor := []Entry{
		{"meta": meta},
		{"component": componentSpec},
	}

	if err := deepSort(normalisedComponentDescriptor); err != nil {
		return nil, fmt.Errorf("unable to sort normalised component descriptor: %w", err)
	}

	byteBuffer := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteBuffer)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(normalisedComponentDescriptor); err != nil {
		return nil, err
	}

	normalisedJson := byteBuffer.Bytes()

	// encoder.Encode appends a newline that we do not want
	if normalisedJson[len(normalisedJson)-1] == 10 {
		normalisedJson = normalisedJson[:len(normalisedJson)-1]
	}

	return normalisedJson, nil
}

func buildExtraIdentity(identity cdv2.Identity) []Entry {
	var extraIdentities []Entry
	for k, v := range identity {
		extraIdentities = append(extraIdentities, Entry{k: v})
	}
	return extraIdentities
}

// deepSort sorts Entry, []Enry and [][]Entry interfaces recursively, lexicographicly by key(Entry).
func deepSort(in interface{}) error {
	switch castIn := in.(type) {
	case []Entry:
		// sort the values recursively for every entry
		for _, entry := range castIn {
			val := getOnlyValueInEntry(entry)
			if err := deepSort(val); err != nil {
				return err
			}
		}
		// sort the entries based on the key
		sort.SliceStable(castIn, func(i, j int) bool {
			return getOnlyKeyInEntry(castIn[i]) < getOnlyKeyInEntry(castIn[j])
		})
	case Entry:
		val := getOnlyValueInEntry(castIn)
		if err := deepSort(val); err != nil {
			return err
		}
	case []interface{}:
		for _, v := range castIn {
			if err := deepSort(v); err != nil {
				return err
			}
		}
	case string:
		break
	case cdv2.ProviderType:
		break
	case cdv2.ResourceRelation:
		break
	default:
		return fmt.Errorf("unknown type in sorting: %T", in)
	}
	return nil
}

func getOnlyKeyInEntry(entry Entry) string {
	var key string
	for k := range entry {
		key = k
	}
	return key
}

func getOnlyValueInEntry(entry Entry) interface{} {
	var value interface{}
	for _, v := range entry {
		value = v
	}
	return value
}

// isNormaliseable checks if componentReferences and resources contain digest.
// Resources are allowed to omit the digest, if res.access.type == None or res.access == nil.
// Does NOT verify if the digests are correct
func isNormaliseable(cd cdv2.ComponentDescriptor) error {
	// check for digests on component references
	for _, reference := range cd.ComponentReferences {
		if reference.Digest == nil || reference.Digest.HashAlgorithm == "" || reference.Digest.NormalisationAlgorithm == "" || reference.Digest.Value == "" {
			return fmt.Errorf("missing digest in component reference %s:%s", reference.Name, reference.Version)
		}
	}
	for _, res := range cd.Resources {
		if (res.Access != nil && res.Access.Type != "None") && res.Digest == nil {
			return fmt.Errorf("missing digest in resource %s:%s", res.Name, res.Version)
		}
		if (res.Access == nil || res.Access.Type == "None") && res.Digest != nil {
			return fmt.Errorf("digest with emtpy (None) access not allowed in resource %s:%s", res.Name, res.Version)
		}
	}
	return nil
}
