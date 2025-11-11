// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ctfutils

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
)

// ResolveList resolves all component descriptors of a given root component descriptor.
func ResolveList(ctx context.Context,
	resolver ctf.ComponentResolver,
	repoCtx cdv2.Repository,
	name,
	version string) (*cdv2.ComponentDescriptorList, error) {

	list := &cdv2.ComponentDescriptorList{}
	err := ResolveRecursive(ctx, resolver, repoCtx, name, version, func(cd *cdv2.ComponentDescriptor) (stop bool, err error) {
		if _, err := list.GetComponent(cd.Name, cd.Version); err != nil {
			list.Components = append(list.Components, *cd)
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ResolvedCallbackFunc describes a function that is called when a component descriptor is resolved.
// The function can optionally return an bool which when set to true stops the resolve of further component descriptors
type ResolvedCallbackFunc func(descriptor *cdv2.ComponentDescriptor) (stop bool, err error)

// ResolveRecursive recursively resolves all component descriptors dependencies.
// Everytime a new component descriptor is resolved the given callback function is called.
// The resolve of further components can be stopped when
// - the callback returns true for the stop parameter
// - the callback returns an error
// - all components are successfully resolved.
func ResolveRecursive(ctx context.Context, resolver ctf.ComponentResolver, repoCtx cdv2.Repository, name, version string, cb ResolvedCallbackFunc) error {
	cd, err := resolver.Resolve(ctx, repoCtx, name, version)
	if err != nil {
		return fmt.Errorf("unable to resolve component descriptor for %q %q %q: %w", repoCtx.GetType(), name, version, err)
	}
	stop, err := cb(cd)
	if err != nil {
		return fmt.Errorf("error while calling callback for %q %q %q: %w", repoCtx.GetType(), name, version, err)
	}
	if stop {
		return nil
	}
	return resolveRecursive(ctx, resolver, repoCtx, cd, cb)
}

func resolveRecursive(ctx context.Context, resolver ctf.ComponentResolver, repoCtx cdv2.Repository, cd *cdv2.ComponentDescriptor, cb ResolvedCallbackFunc) error {
	components := make([]*cdv2.ComponentDescriptor, 0)
	for _, ref := range cd.ComponentReferences {
		cd, err := resolver.Resolve(ctx, repoCtx, ref.ComponentName, ref.Version)
		if err != nil {
			return fmt.Errorf("unable to resolve component descriptor for %q %q %q: %w", repoCtx.GetType(), ref.ComponentName, ref.Version, err)
		}
		components = append(components, cd)
		stop, err := cb(cd)
		if err != nil {
			return fmt.Errorf("error while calling callback for %q %q %q: %w", repoCtx.GetType(), ref.ComponentName, ref.Version, err)
		}
		if stop {
			return nil
		}
	}
	for _, ref := range components {
		if err := resolveRecursive(ctx, resolver, repoCtx, ref, cb); err != nil {
			return err
		}
	}
	return nil
}
