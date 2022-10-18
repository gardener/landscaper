// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// ReferenceDiff returns a human readable diff for two refs
func ReferenceDiff(oldRef, newRef *lsv1alpha1.ComponentDescriptorReference) string {
	var (
		repoCtx       = "RepositoryContext not updated"
		componentName = "ComponentName not updated"
		version       = "Version not updated"
	)

	if !cdv2.UnstructuredTypesEqual(oldRef.RepositoryContext, newRef.RepositoryContext) {
		repoCtx = repositoryContextDiff(oldRef.RepositoryContext, newRef.RepositoryContext)
	}
	if oldRef.ComponentName != newRef.ComponentName {
		componentName = fmt.Sprintf("%s -> %s", oldRef.ComponentName, newRef.ComponentName)
	}
	if oldRef.Version != newRef.Version {
		componentName = fmt.Sprintf("%s -> %s", oldRef.Version, newRef.Version)
	}
	return fmt.Sprintf(`
Componenten reference has been overwritten:
%s
%s
%s
`, repoCtx, componentName, version)
}

func repositoryContextDiff(oldCtx, newCtx *cdv2.UnstructuredTypedObject) string {
	if oldCtx == nil {
		return "-> " + string(newCtx.Raw)
	}
	defaultDiff := fmt.Sprintf("%s -> %s", string(oldCtx.Raw), string(newCtx.Raw))
	// use specific behavior if both repository contexts are ociregistries
	if oldCtx.GetType() == cdv2.OCIRegistryType &&
		newCtx.GetType() == cdv2.OCIRegistryType {
		oldOciReg := cdv2.OCIRegistryRepository{}
		if err := oldCtx.DecodeInto(&oldOciReg); err != nil {
			return defaultDiff
		}
		newOciReg := cdv2.OCIRegistryRepository{}
		if err := newCtx.DecodeInto(&newOciReg); err != nil {
			return defaultDiff
		}
		return fmt.Sprintf("%s (%s) -> %s (%s)", oldOciReg.BaseURL, oldOciReg.ComponentNameMapping, newOciReg.BaseURL, newOciReg.ComponentNameMapping)
	}
	return defaultDiff
}

var _ ctf.ComponentResolver = overwriteResolver{}

type overwriteResolver struct {
	overwriter Overwriter
	resolver   ctf.ComponentResolver
}

// OverwriteResolver returns a new OverwriteResolver
// OverwriteResolver is a ctf.ComponentResolver, which applies the given overwrites before resolving with the given resolver.
func OverwriteResolver(resolver ctf.ComponentResolver, overwriter Overwriter) overwriteResolver {
	return overwriteResolver{
		overwriter: overwriter,
		resolver:   resolver,
	}
}

// toCDRef converts a repository, name, and version into a component descriptor reference
func toCDRef(repoCtx cdv2.Repository, name, version string) (*lsv1alpha1.ComponentDescriptorReference, error) {
	cdRef := &lsv1alpha1.ComponentDescriptorReference{
		ComponentName: name,
		Version:       version,
	}
	if repoCtx != nil {
		repoRefConverted, err := cdv2.NewUnstructured(repoCtx)
		if err != nil {
			return nil, err
		}
		cdRef.RepositoryContext = &repoRefConverted
	}
	return cdRef, nil
}

func (or overwriteResolver) Resolve(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, error) {
	if or.overwriter != nil {
		cdRef, err := toCDRef(repoCtx, name, version)
		if err != nil {
			return nil, err
		}
		or.overwriter.Replace(cdRef)
		return or.resolver.Resolve(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	}
	return or.resolver.Resolve(ctx, repoCtx, name, version)
}

func (or overwriteResolver) ResolveWithBlobResolver(ctx context.Context, repoCtx cdv2.Repository, name, version string) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	if or.overwriter != nil {
		cdRef, err := toCDRef(repoCtx, name, version)
		if err != nil {
			return nil, nil, err
		}
		or.overwriter.Replace(cdRef)
		return or.resolver.ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
	}
	return or.resolver.ResolveWithBlobResolver(ctx, repoCtx, name, version)
}
