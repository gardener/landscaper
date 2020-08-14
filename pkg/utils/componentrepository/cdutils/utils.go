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

package cdutils

import (
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// FindResourceByVersionedReference searches all given components for the defined resource ref.
func FindResourceByVersionedReference(ttype string, ref lsv1alpha1.VersionedResourceReference, components ...cdv2.ComponentDescriptor) (cdv2.Resource, error) {
	for _, comp := range components {
		res, err := FindResourceInComponentByVersionedReference(comp, ttype, ref)
		if !errors.Is(err, cdv2.NotFound) {
			return cdv2.Resource{}, err
		}
		if err == nil {
			return res, nil
		}
	}
	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceInComponentByVersionedReference searches the given component for the defined resource ref.
func FindResourceInComponentByVersionedReference(comp cdv2.ComponentDescriptor, ttype string, ref lsv1alpha1.VersionedResourceReference) (cdv2.Resource, error) {
	if comp.GetName() != ref.ComponentName {
		return cdv2.Resource{}, cdv2.NotFound
	}
	if comp.GetVersion() != ref.Version {
		return cdv2.Resource{}, cdv2.NotFound
	}

	if ref.Kind != lsv1alpha1.LocalResourceKind && ref.Kind != lsv1alpha1.ExternalResourceKind {
		return cdv2.Resource{}, fmt.Errorf("unexpected resource kind %s: %w", ref.Kind, lsv1alpha1.UnknownResourceKindError)
	}

	if ref.Kind == lsv1alpha1.LocalResourceKind {
		res, err := comp.GetLocalResource(ttype, ref.Resource, ref.Version)
		if err != nil {
			return cdv2.Resource{}, err
		}
		return res, nil
	}

	if ref.Kind == lsv1alpha1.ExternalResourceKind {
		res, err := comp.GetExternalResource(ttype, ref.Resource, ref.Version)
		if err != nil {
			return cdv2.Resource{}, err
		}
		return res, nil
	}

	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceByReference searches all given components for the defined resource ref.
func FindResourceByReference(ttype string, ref lsv1alpha1.ResourceReference, components ...cdv2.ComponentDescriptor) (cdv2.Resource, error) {
	for _, comp := range components {
		res, err := FindResourceInComponentByReference(comp, ttype, ref)
		if !errors.Is(err, cdv2.NotFound) {
			return cdv2.Resource{}, err
		}
		if err == nil {
			return res, nil
		}
	}
	return cdv2.Resource{}, cdv2.NotFound
}

// FindResourceInComponentByReference searches the given component for the defined resource ref.
func FindResourceInComponentByReference(comp cdv2.ComponentDescriptor, ttype string, ref lsv1alpha1.ResourceReference) (cdv2.Resource, error) {
	if comp.GetName() != ref.ComponentName {
		return cdv2.Resource{}, cdv2.NotFound
	}

	if ref.Kind != lsv1alpha1.LocalResourceKind && ref.Kind != lsv1alpha1.ExternalResourceKind {
		return cdv2.Resource{}, fmt.Errorf("unexpected resource kind %s: %w", ref.Kind, lsv1alpha1.UnknownResourceKindError)
	}

	if ref.Kind == lsv1alpha1.LocalResourceKind {
		resources := comp.GetLocalResourcesByName(ttype, ref.Resource)
		if len(resources) == 0 {
			return cdv2.Resource{}, cdv2.NotFound
		}
		return resources[0], nil
	}

	if ref.Kind == lsv1alpha1.ExternalResourceKind {
		resources := comp.GetExternalResourcesByName(ttype, ref.Resource)
		if len(resources) == 0 {
			return cdv2.Resource{}, cdv2.NotFound
		}
		return resources[0], nil
	}

	return cdv2.Resource{}, cdv2.NotFound
}
