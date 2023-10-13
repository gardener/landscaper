// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/utils/selector"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/mediatype"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/utils"
)

// TODO: investigate if this coding can be removed entirely after component-cli is removed

// ResolveBlueprint returns a blueprint from a given reference.
// If no fs is given, a temporary filesystem will be created.
func ResolveBlueprint(ctx context.Context,
	registry model.RegistryAccess,
	cdRef *lsv1alpha1.ComponentDescriptorReference,
	bpDef lsv1alpha1.BlueprintDefinition) (*Blueprint, error) {

	if bpDef.Reference == nil && bpDef.Inline == nil {
		return nil, errors.New("no remote reference nor a inline blueprint is defined")
	}

	if bpDef.Inline != nil {
		// todo: check if it is necessary to write it to disk.
		// although it is already in memory though the installation.
		fs := memoryfs.New()
		inlineFs, err := yamlfs.New(bpDef.Inline.Filesystem.RawMessage)
		if err != nil {
			return nil, fmt.Errorf("unable to create yamlfs for inline blueprint: %w", err)
		}
		if err := utils.CopyFS(inlineFs, fs, "/", "/"); err != nil {
			return nil, fmt.Errorf("unable to copy yaml filesystem: %w", err)
		}
		// read blueprint yaml from file system
		data, err := vfs.ReadFile(fs, lsv1alpha1.BlueprintFileName)
		if err != nil {
			return nil, fmt.Errorf("unable to read blueprint file from inline defined blueprint: %w", err)
		}
		blue := &lsv1alpha1.Blueprint{}
		if _, _, err := serializer.NewCodecFactory(api.LandscaperScheme).UniversalDecoder().Decode(data, nil, blue); err != nil {
			return nil, fmt.Errorf("unable to decode blueprint definition from inline defined blueprint. %w", err)
		}
		return New(blue, readonlyfs.New(fs)), nil
	}

	if cdRef == nil {
		return nil, fmt.Errorf("no component descriptor reference defined")
	}
	if cdRef.RepositoryContext == nil {
		return nil, fmt.Errorf("no respository context defined")
	}
	if registry == nil {
		return nil, fmt.Errorf("did not get a working component descriptor resolver")
	}

	componentVersion, err := registry.GetComponentVersion(ctx, cdRef)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}
	resource, err := componentVersion.GetResource(bpDef.Reference.ResourceName, nil)
	if err != nil {
		return nil, err
	}
	content, err := resource.GetTypedContent(ctx)
	if err != nil {
		return nil, err
	}
	blueprint, ok := content.Resource.(*Blueprint)
	if !ok {
		return nil, fmt.Errorf("received resource of type %T but expected type *Blueprint", blueprint)
	}
	return blueprint, nil
}

// Resolve returns a blueprint from a given reference.
// If no fs is given, a temporary filesystem will be created.
func Resolve(ctx context.Context, registryAccess model.RegistryAccess, cdRef *lsv1alpha1.ComponentDescriptorReference, bpDef lsv1alpha1.BlueprintDefinition) (*Blueprint, error) {
	if bpDef.Reference == nil && bpDef.Inline == nil {
		return nil, errors.New("no remote reference nor a inline blueprint is defined")
	}

	if bpDef.Inline != nil {
		// todo: check if it is necessary to write it to disk.
		// although it is already in memory though the installation.
		fs := memoryfs.New()
		inlineFs, err := yamlfs.New(bpDef.Inline.Filesystem.RawMessage)
		if err != nil {
			return nil, fmt.Errorf("unable to create yamlfs for inline blueprint: %w", err)
		}
		if err := utils.CopyFS(inlineFs, fs, "/", "/"); err != nil {
			return nil, fmt.Errorf("unable to copy yaml filesystem: %w", err)
		}
		// read blueprint yaml from file system
		data, err := vfs.ReadFile(fs, lsv1alpha1.BlueprintFileName)
		if err != nil {
			return nil, fmt.Errorf("unable to read blueprint file from inline defined blueprint: %w", err)
		}
		blue := &lsv1alpha1.Blueprint{}
		if _, _, err := serializer.NewCodecFactory(api.LandscaperScheme).UniversalDecoder().Decode(data, nil, blue); err != nil {
			return nil, fmt.Errorf("unable to decode blueprint definition from inline defined blueprint. %w", err)
		}
		return New(blue, readonlyfs.New(fs)), nil
	}

	if cdRef == nil {
		return nil, fmt.Errorf("no component descriptor reference defined")
	}
	if cdRef.RepositoryContext == nil {
		return nil, fmt.Errorf("no respository context defined")
	}
	if registryAccess == nil {
		return nil, fmt.Errorf("did not get a working component descriptor resolver")
	}
	componentVersion, err := registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
		RepositoryContext: cdRef.RepositoryContext,
		ComponentName:     cdRef.ComponentName,
		Version:           cdRef.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}
	resource, err := componentVersion.GetResource(bpDef.Reference.ResourceName, nil)
	if err != nil {
		return nil, err
	}
	content, err := resource.GetTypedContent(ctx)
	if err != nil {
		return nil, err
	}
	blueprint, ok := content.Resource.(*Blueprint)
	if !ok {
		return nil, fmt.Errorf("received resource of type %T but expected type *Blueprint", blueprint)
	}
	return blueprint, nil
}

// GetBlueprintResourceFromComponentDescriptor returns the blueprint resource from a component descriptor.
func GetBlueprintResourceFromComponentDescriptor(cd *types.ComponentDescriptor, blueprintName string) (types.Resource, error) {
	// get blueprint resource from component descriptor
	resources, err := cd.GetResourcesByType(mediatype.BlueprintType, selector.DefaultSelector{cdv2.SystemIdentityName: blueprintName})
	if err != nil {
		if !errors.Is(err, cdv2.NotFound) {
			return types.Resource{}, fmt.Errorf("unable to find blueprint %s in component descriptor: %w", blueprintName, err)
		}
		// try to fallback to old blueprint type
		resources, err = cd.GetResourcesByType(mediatype.OldBlueprintType, selector.DefaultSelector{cdv2.SystemIdentityName: blueprintName})
		if err != nil {
			return types.Resource{}, fmt.Errorf("unable to find blueprint %s in component descriptor: %w", blueprintName, err)
		}
	}
	// todo: consider to throw an error if multiple blueprints match
	return resources[0], nil
}

// GetBlueprintResourceFromComponentVersion returns the blueprint resource from a component descriptor.
func GetBlueprintResourceFromComponentVersion(componentVersion model.ComponentVersion, blueprintName string) (model.Resource, error) {
	resource, err := componentVersion.GetResource(blueprintName, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get blueprint %s from component descriptor: %w", blueprintName, err)
	}

	if resource.GetType() != mediatype.BlueprintType && resource.GetType() != mediatype.OldBlueprintType {
		return nil, fmt.Errorf("blueprint resource %s has wrong type %s", blueprintName, resource.GetType())
	}

	return resource, nil
}
