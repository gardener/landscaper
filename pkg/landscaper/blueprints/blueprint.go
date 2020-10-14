// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"fmt"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

// Blueprint is the internal resolved type of a blueprint.
type Blueprint struct {
	Info             *lsv1alpha1.Blueprint
	Fs               vfs.FileSystem
	Subinstallations []*lsv1alpha1.InstallationTemplate
}

// New creates a new internal Blueprint from a blueprint definition and its filesystem content.
func New(blueprint *lsv1alpha1.Blueprint, content vfs.FileSystem) (*Blueprint, error) {
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
	refs := make([]*lsv1alpha1.InstallationTemplate, len(blueprint.Info.Subinstallations))
	for i, subInstTmpl := range blueprint.Info.Subinstallations {
		if subInstTmpl.InstallationTemplate != nil {
			refs[i] = subInstTmpl.InstallationTemplate
			continue
		}

		if len(subInstTmpl.File) == 0 {
			return fmt.Errorf("neither a inline installtion template nor a file is defined in index %d", i)
		}
		data, err := vfs.ReadFile(blueprint.Fs, subInstTmpl.File)
		if err != nil {
			return err
		}

		instTmpl := &lsv1alpha1.InstallationTemplate{}
		if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, instTmpl); err != nil {
			return err
		}
		refs[i] = instTmpl
	}

	blueprint.Subinstallations = refs
	return nil
}

//// RemoteBlueprintReference returns the remote blueprint ref for the current component given the effective component descriptor
//func (r InstallationTemplate) RemoteBlueprintReference(cdList cdv2.ComponentDescriptorList) (lsv1alpha1.BlueprintDefinition, error) {
//	components := cdList.GetComponentByName(r.Info.Reference.ComponentName)
//	if len(components) == 0 {
//		return lsv1alpha1.BlueprintDefinition{}, cdv2.NotFound
//	}
//
//	res, err := cdutils.FindResourceInComponentByReference(components[0], lsv1alpha1.BlueprintResourceType, r.Info.Reference)
//	if err != nil {
//		return lsv1alpha1.BlueprintDefinition{}, cdv2.NotFound
//	}
//
//	repoCtx := components[0].GetEffectiveRepositoryContext()
//	return lsv1alpha1.BlueprintDefinition{
//		Reference: &lsv1alpha1.RemoteBlueprintReference{
//			RepositoryContext: &repoCtx,
//			VersionedResourceReference: lsv1alpha1.VersionedResourceReference{
//				ResourceReference: r.Info.Reference,
//				Version:           res.Version,
//			},
//		},
//	}, nil
//}
