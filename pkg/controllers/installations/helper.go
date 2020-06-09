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

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/component"
	"github.com/gardener/landscaper/pkg/landscaper/registry"
	"github.com/gardener/landscaper/pkg/utils"
)

var componentInstallationGVK schema.GroupVersionKind

func init() {
	var err error
	componentInstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.ComponentInstallation{}, kubernetes.LandscaperScheme)
	runtime.Must(err)
}

// IsRootInstallation returns if the installation is a root element.
func IsRootInstallation(inst *lsv1alpha1.ComponentInstallation) bool {
	_, isOwned := utils.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return !isOwned
}

// GetParentInstallationName returns the name of parent installation that encompasses the given installation.
func GetParentInstallationName(inst *lsv1alpha1.ComponentInstallation) string {
	name, _ := utils.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return name
}

// GetRootInstallations returns all root installations in the system
func (a *actuator) GetRootInstallations(ctx context.Context, opts ...client.ListOption) ([]*lsv1alpha1.ComponentInstallation, error) {
	r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)})

	installationList := &lsv1alpha1.ComponentInstallationList{}
	if err := a.c.List(ctx, installationList, opts...); err != nil {
		return nil, err
	}

	installations := make([]*lsv1alpha1.ComponentInstallation, len(installationList.Items))
	for i, obj := range installationList.Items {
		inst := obj
		installations[i] = &inst
	}
	return installations, nil
}

// CreateInternalInstallations creates internal installations for a list of ComponentInstallations
func CreateInternalInstallations(registry registry.Registry, installations ...*lsv1alpha1.ComponentInstallation) ([]*component.Installation, error) {
	internalInstallations := make([]*component.Installation, len(installations))
	for i, inst := range installations {
		inInst, err := CreateInternalInstallation(registry, inst)
		if err != nil {
			return nil, err
		}
		internalInstallations[i] = inInst
	}
	return internalInstallations, nil
}

// CreateInternalInstallation creates an internal installation for a ComponentInstallation
func CreateInternalInstallation(registry registry.Registry, inst *lsv1alpha1.ComponentInstallation) (*component.Installation, error) {
	def, err := registry.GetDefinitionByRef(inst.Spec.DefinitionRef)
	if err != nil {
		return nil, err
	}
	return component.New(inst, def)
}

// AddDefaultMappings adds all default mappings of im and exports if they are not already defined
func AddDefaultMappings(inst *lsv1alpha1.ComponentInstallation, def *lsv1alpha1.ComponentDefinition) {
	mappings := sets.NewString()
	for _, mapping := range inst.Spec.Imports {
		mappings.Insert(mapping.To)
	}
	for _, importDef := range def.Imports {
		if !mappings.Has(importDef.Key) {
			inst.Spec.Imports = append(inst.Spec.Imports, lsv1alpha1.DefinitionImportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
					From: importDef.Key,
					To:   importDef.Key,
				},
			})
		}
	}

	mappings = sets.NewString()
	for _, mapping := range inst.Spec.Exports {
		mappings.Insert(mapping.From)
	}
	for _, importDef := range def.Exports {
		if !mappings.Has(importDef.Key) {
			inst.Spec.Exports = append(inst.Spec.Exports, lsv1alpha1.DefinitionExportMapping{
				DefinitionFieldMapping: lsv1alpha1.DefinitionFieldMapping{
					From: importDef.Key,
					To:   importDef.Key,
				},
			})
		}
	}
}
