// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints

import (
	"context"
	"errors"
	"fmt"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/readonlyfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// Resolve returns a blueprint from a given reference.
// If no fs is given, a temporary filesystem will be created.
func Resolve(ctx context.Context, op operation.RegistriesAccessor, def lsv1alpha1.BlueprintDefinition, fs vfs.FileSystem) (*Blueprint, error) {
	if def.Reference == nil && def.Inline == nil {
		return nil, errors.New("no remote reference nor a inline blueprint is defined")
	}

	if def.Inline != nil {
		fs, err := yamlfs.New(def.Inline.Filesystem)
		if err != nil {
			return nil, fmt.Errorf("unable to create yamlfs for inline blueprint: %w", err)
		}
		// read blueprint yaml from file system
		data, err := vfs.ReadFile(fs, lsv1alpha1.BlueprintFilePath)
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

	reference := def.Reference
	if reference.RepositoryContext == nil {
		return nil, fmt.Errorf("no respository context defined")
	}
	cd, err := op.ComponentsRegistry().Resolve(ctx, *reference.RepositoryContext, reference.ObjectMeta())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve component descriptor for ref %#v: %w", reference, err)
	}

	res, err := cdutils.FindResourceInComponentByVersionedReference(*cd, lsv1alpha1.BlueprintResourceType, reference.VersionedResourceReference)
	if err != nil {
		return nil, fmt.Errorf("unable to find blueprint resource in component descriptor for ref %#v: %w", reference, err)
	}

	blue, err := op.BlueprintsRegistry().GetBlueprint(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch blueprint for ref %#v: %w", reference, err)
	}

	if fs == nil {
		osFs := osfs.New()
		tmpDir, err := vfs.TempDir(osFs, osFs.FSTempDir(), "blueprint-")
		if err != nil {
			return nil, fmt.Errorf("unable to create temporary directory: %w", err)
		}
		fs, err = projectionfs.New(osFs, tmpDir)
		if err != nil {
			return nil, fmt.Errorf("unable to create virtual filesystem with base path %s for content for ref %#v: %w", tmpDir, reference, err)
		}

	}
	if err := op.BlueprintsRegistry().GetContent(ctx, res, fs); err != nil {
		return nil, fmt.Errorf("unable to fetch content for ref %#v: %w", reference, err)
	}

	intBlueprint, err := New(blue, readonlyfs.New(fs))
	if err != nil {
		return nil, fmt.Errorf("unable to create internal blueprint representation for ref %#v: %w", reference, err)
	}
	return intBlueprint, nil
}
