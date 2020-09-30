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
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	lscheme "github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

var componentInstallationGVK schema.GroupVersionKind

func init() {
	var err error
	componentInstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.Installation{}, lscheme.LandscaperScheme)
	runtime.Must(err)
}

// IsRootInstallation returns if the installation is a root element.
func IsRootInstallation(inst *lsv1alpha1.Installation) bool {
	_, isOwned := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return !isOwned
}

// GetParentInstallationName returns the name of parent installation that encompasses the given installation.
func GetParentInstallationName(inst *lsv1alpha1.Installation) string {
	name, _ := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return name
}

// CreateInternalInstallations creates internal installations for a list of ComponentInstallations
func CreateInternalInstallations(ctx context.Context, op lsoperation.Interface, installations ...*lsv1alpha1.Installation) ([]*Installation, error) {
	internalInstallations := make([]*Installation, len(installations))
	for i, inst := range installations {
		inInst, err := CreateInternalInstallation(ctx, op, inst)
		if err != nil {
			return nil, err
		}
		internalInstallations[i] = inInst
	}
	return internalInstallations, nil
}

// ResolveComponentDescriptor resolves the component descriptor of a installation.
func ResolveComponentDescriptor(ctx context.Context, compRepo componentsregistry.Registry, inst *lsv1alpha1.Installation) (*cdv2.ComponentDescriptor, error) {
	if inst.Spec.Blueprint.Reference == nil &&
		(inst.Spec.Blueprint.Inline == nil || inst.Spec.Blueprint.Inline.ComponentDescriptorReference == nil) {
		return nil, nil
	}
	var (
		repoCtx cdv2.RepositoryContext
		ref     cdv2.ObjectMeta
	)
	if inst.Spec.Blueprint.Reference != nil {
		// todo: if not defined read from default configured repo context.
		repoCtx = *inst.Spec.Blueprint.Reference.RepositoryContext
		ref = inst.Spec.Blueprint.Reference.ObjectMeta()
	}
	if inst.Spec.Blueprint.Inline != nil && inst.Spec.Blueprint.Inline.ComponentDescriptorReference != nil {
		repoCtx = *inst.Spec.Blueprint.Inline.ComponentDescriptorReference.RepositoryContext
		ref = inst.Spec.Blueprint.Inline.ComponentDescriptorReference.ObjectMeta()
	}
	return compRepo.Resolve(ctx, repoCtx, ref)
}

// CreateInternalInstallation creates an internal installation for a Installation
func CreateInternalInstallation(ctx context.Context, op lsoperation.Interface, inst *lsv1alpha1.Installation) (*Installation, error) {
	blue, err := blueprints.Resolve(ctx, op, inst.Spec.Blueprint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}
