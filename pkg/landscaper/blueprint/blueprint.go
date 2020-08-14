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

package blueprint

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils/componentrepository/cdutils"
)

// Blueprint is the internal resolved type of a blueprint.
type Blueprint struct {
	Info       *lsv1alpha1.Blueprint
	Fs         afero.Fs
	References []*BlueprintReference
}

// BlueprintReference is the internal type of a blueprint reference.
type BlueprintReference struct {
	Reference *lsv1alpha1.BlueprintReference
	Path      string
	Fs        afero.Fs
}

// New creates a new internal Blueprint from a blueprint definition and its filesystem content.
func New(blueprint *lsv1alpha1.Blueprint, content afero.Fs) (*Blueprint, error) {
	b := &Blueprint{
		Info: blueprint,
		Fs:   content,
	}

	if err := ResolveBlueprintReferences(b); err != nil {
		return nil, err
	}

	return b, nil
}

func ResolveBlueprintReferences(blueprint *Blueprint) error {
	refs := make([]*BlueprintReference, len(blueprint.Info.BlueprintReferences))
	for i, path := range blueprint.Info.BlueprintReferences {
		data, err := afero.ReadFile(blueprint.Fs, path)
		if err != nil {
			return err
		}

		blueprintRef := &lsv1alpha1.BlueprintReference{}
		if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, blueprintRef); err != nil {
			return err
		}
		refs[i] = &BlueprintReference{
			Reference: blueprintRef,
			Path:      path,
			Fs:        blueprint.Fs,
		}
	}

	blueprint.References = refs
	return nil
}

// RemoteBlueprintReference returns the remote blueprint ref for the current component given the effective component descriptor
func (r BlueprintReference) RemoteBlueprintReference(cdList cdv2.ComponentDescriptorList) (lsv1alpha1.RemoteBlueprintReference, error) {
	components := cdList.GetComponentByName(r.Reference.Reference.ComponentName)
	if len(components) == 0 {
		return lsv1alpha1.RemoteBlueprintReference{}, cdv2.NotFound
	}

	res, err := cdutils.FindResourceInComponentByReference(components[0], lsv1alpha1.BlueprintResourceType, r.Reference.Reference)
	if err != nil {
		return lsv1alpha1.RemoteBlueprintReference{}, cdv2.NotFound
	}

	return lsv1alpha1.RemoteBlueprintReference{
		RepositoryContext: components[0].GetEffectiveRepositoryContext(),
		VersionedResourceReference: lsv1alpha1.VersionedResourceReference{
			ResourceReference: r.Reference.Reference,
			Version:           res.Version,
		},
	}, nil
}
