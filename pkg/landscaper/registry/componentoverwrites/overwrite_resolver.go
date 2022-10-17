// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentoverwrites

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

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
