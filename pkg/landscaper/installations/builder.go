// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/components/model"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
)

// OperationBuilder is a builder helper struct for building an installation operation.
type OperationBuilder struct {
	lsoperation.Builder

	inst                            *InstallationImportsAndBlueprint
	componentVersion                model.ComponentVersion
	op                              *lsoperation.Operation
	resolvedComponentDescriptorList *model.ComponentVersionList
	context                         *Scope
}

// NewOperationBuilder creates a new operation builder.
func NewOperationBuilder(inst *InstallationImportsAndBlueprint) *OperationBuilder {
	return &OperationBuilder{
		inst: inst,
	}
}

// Installation sets an installation.
func (b *OperationBuilder) Installation(inst *InstallationImportsAndBlueprint) *OperationBuilder {
	b.inst = inst
	return b
}

// ComponentVersion sets the component version for the builder.
// Will be calculated if not set.
func (b *OperationBuilder) ComponentVersion(componentVersion model.ComponentVersion) *OperationBuilder {
	b.componentVersion = componentVersion
	return b
}

// WithComponentDescriptorList sets the list of transitive component descriptors.
// Will be calculated if not set.
func (b *OperationBuilder) WithComponentDescriptorList(list *model.ComponentVersionList) *OperationBuilder {
	b.resolvedComponentDescriptorList = list
	return b
}

// WithOperation sets the base operation.
func (b *OperationBuilder) WithOperation(op *lsoperation.Operation) *OperationBuilder {
	b.op = op
	return b
}

// WithContext sets an optional context.
// This value will be calculated during the build if not set.
func (b *OperationBuilder) WithContext(ctx *Scope) *OperationBuilder {
	b.context = ctx
	return b
}

// operation builder wrapped options

// Client sets the kubernetes client.
func (b *OperationBuilder) WithLsUncachedClient(lsUncachedClient client.Client) *OperationBuilder {
	b.Builder.WithLsUncachedClient(lsUncachedClient)
	return b
}

// Scheme sets the kubernetes scheme.
func (b *OperationBuilder) Scheme(s *runtime.Scheme) *OperationBuilder {
	b.Builder.Scheme(s)
	return b
}

// ComponentRegistry sets the component registry.
func (b *OperationBuilder) ComponentRegistry(registry model.RegistryAccess) *OperationBuilder {
	b.Builder.ComponentRegistry(registry)
	return b
}

// WithEventRecorder sets a event recorder.
func (b *OperationBuilder) WithEventRecorder(er record.EventRecorder) *OperationBuilder {
	b.Builder.WithEventRecorder(er)
	return b
}

func (b *OperationBuilder) validate() error {
	if b.inst == nil {
		return errors.New("a installation must be set")
	}
	return nil
}

// Build creates an installation operation.
func (b *OperationBuilder) Build(ctx context.Context) (*Operation, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if b.op == nil {
		// try to build a new operation
		op, err := b.Builder.Build(ctx)
		if err != nil {
			return nil, err
		}
		b.op = op
	}

	instOp := &Operation{
		Operation:                       b.op,
		Inst:                            b.inst,
		ComponentVersion:                b.componentVersion,
		ResolvedComponentDescriptorList: b.resolvedComponentDescriptorList,
	}

	if b.context == nil {
		newCtx, err := GetInstallationContext(ctx, instOp.LsUncachedClient(), instOp.Inst.GetInstallation())
		if err != nil {
			return nil, err
		}
		b.context = newCtx
	}
	instOp.context = *b.context

	if instOp.ComponentVersion == nil {
		registryAccess := instOp.ComponentsRegistry()
		cdRef := instOp.Context().External.ComponentDescriptorRef()
		if cdRef == nil || registryAccess == nil {
			return instOp, nil
		}

		componentVersion, err := registryAccess.GetComponentVersion(ctx, cdRef)
		if err != nil {
			return nil, err
		}

		instOp.ComponentVersion = componentVersion
	}

	if instOp.ResolvedComponentDescriptorList == nil {
		componentVersions, err := model.GetTransitiveComponentReferences(ctx,
			instOp.ComponentVersion,
			instOp.Context().External.RepositoryContext,
			instOp.Context().External.Overwriter)
		if err != nil {
			return nil, err
		}

		instOp.ResolvedComponentDescriptorList = componentVersions
	}

	return instOp, nil
}
