// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/gardener/component-spec/bindings-go/utils/selector"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils"
)

// Resolve returns a blueprint from a given reference.
// If no fs is given, a temporary filesystem will be created.
func Resolve(ctx context.Context, resolver ctf.ComponentResolver, cdRef *lsv1alpha1.ComponentDescriptorReference, bpDef lsv1alpha1.BlueprintDefinition, fs vfs.FileSystem) (*Blueprint, error) {
	if bpDef.Reference == nil && bpDef.Inline == nil {
		return nil, errors.New("no remote reference nor a inline blueprint is defined")
	}

	if fs == nil {
		osFs := osfs.New()
		tmpDir, err := vfs.TempDir(osFs, osFs.FSTempDir(), "blueprint-")
		if err != nil {
			return nil, fmt.Errorf("unable to create temporary directory: %w", err)
		}
		fs, err = projectionfs.New(osFs, tmpDir)
		if err != nil {
			return nil, fmt.Errorf("unable to create virtual filesystem with base path %s for content: %w", tmpDir, err)
		}
	}

	if bpDef.Inline != nil {
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
		if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, blue); err != nil {
			return nil, fmt.Errorf("unable to decode blueprint definition from inline defined blueprint. %w", err)
		}
		intBlueprint, err := New(blue, readonlyfs.New(fs))
		if err != nil {
			return nil, fmt.Errorf("unable to create internal blueprint representation for inline config: %w", err)
		}
		return intBlueprint, nil
	}

	if cdRef == nil {
		return nil, fmt.Errorf("no component descriptor reference defined")
	}
	if cdRef.RepositoryContext == nil {
		return nil, fmt.Errorf("no respository context defined")
	}
	if resolver == nil {
		return nil, fmt.Errorf("did not get a working component descriptor resolver")
	}
	cd, blobResolver, err := resolver.Resolve(ctx, *cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", cdRef, err)
	}

	blueprint, err := ResolveBlueprintFromBlobResolver(ctx, cd, blobResolver, fs, bpDef.Reference.ResourceName)
	if err != nil {
		return nil, err
	}

	intBlueprint, err := New(blueprint, readonlyfs.New(fs))
	if err != nil {
		return nil, fmt.Errorf("unable to create internal blueprint representation for ref %#v: %w", cdRef, err)
	}
	return intBlueprint, nil
}

// ResolveBlueprintFromBlobResolver resolves a blueprint defined by a component descriptor with
// a blob resolver.
func ResolveBlueprintFromBlobResolver(ctx context.Context,
	cd *cdv2.ComponentDescriptor,
	blobResolver ctf.BlobResolver,
	fs vfs.FileSystem,
	blueprintName string) (*lsv1alpha1.Blueprint, error) {

	// get blueprint resource from component descriptor
	resource, err := GetBlueprintResourceFromComponentDescriptor(cd, blueprintName)
	if err != nil {
		return nil, err
	}

	var data bytes.Buffer
	if _, err := blobResolver.Resolve(ctx, resource, &data); err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint blob: %w", err)
	}
	if err := utils.ExtractTarGzip(&data, fs, "/"); err != nil {
		return nil, fmt.Errorf("unable to extract blueprint from tar.gzip blob: %w", err)
	}

	blueprintBytes, err := vfs.ReadFile(fs, lsv1alpha1.BlueprintFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read blueprint definition: %w", err)
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).
		UniversalDecoder().
		Decode(blueprintBytes, nil, blueprint); err != nil {
		return nil, fmt.Errorf("unable to decode blueprint definition: %w", err)
	}

	return blueprint, err
}

// GetBlueprintResourceFromComponentDescriptor returns the blueprint resource from a component descriptor.
func GetBlueprintResourceFromComponentDescriptor(cd *cdv2.ComponentDescriptor, blueprintName string) (cdv2.Resource, error) {
	// get blueprint resource from component descriptor
	resources, err := cd.GetResourcesByType(lsv1alpha1.BlueprintType, selector.DefaultSelector{cdv2.SystemIdentityName: blueprintName})
	if err != nil {
		if !errors.Is(err, cdv2.NotFound) {
			return cdv2.Resource{}, fmt.Errorf("unable to find blueprint %s in component descriptor: %w", blueprintName, err)
		}
		// try to fallback to old blueprint type
		resources, err = cd.GetResourcesByType(lsv1alpha1.OldBlueprintType, selector.DefaultSelector{cdv2.SystemIdentityName: blueprintName})
		if err != nil {
			return cdv2.Resource{}, fmt.Errorf("unable to find blueprint %s in component descriptor: %w", blueprintName, err)
		}
	}
	// todo: consider to throw an error if multiple blueprints match
	return resources[0], nil
}
