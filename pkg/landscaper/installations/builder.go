// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"errors"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
)

// OperationBuilder is a builder helper struct for building an installation operation.
type OperationBuilder struct {
	lsoperation.Builder

	inst                            *InstallationImportsAndBlueprint
	cd                              *cdv2.ComponentDescriptor
	op                              *lsoperation.Operation
	blobResolver                    ctf.BlobResolver
	resolvedComponentDescriptorList *cdv2.ComponentDescriptorList
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

// ComponentDescriptor sets the component descriptor for the builder.
// Will be calculated if not set.
func (b *OperationBuilder) ComponentDescriptor(cd *cdv2.ComponentDescriptor) *OperationBuilder {
	b.cd = cd
	return b
}

// WithBlobResolver sets the blob resolver for the component descriptor.
// Will be calculated if not set.
func (b *OperationBuilder) WithBlobResolver(resolver ctf.BlobResolver) *OperationBuilder {
	b.blobResolver = resolver
	return b
}

// WithComponentDescriptorList sets the list of transitive component descriptors.
// Will be calculated if not set.
func (b *OperationBuilder) WithComponentDescriptorList(list *cdv2.ComponentDescriptorList) *OperationBuilder {
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
func (b *OperationBuilder) Client(c client.Client) *OperationBuilder {
	b.Builder.Client(c)
	return b
}

// Scheme sets the kubernetes scheme.
func (b *OperationBuilder) Scheme(s *runtime.Scheme) *OperationBuilder {
	b.Builder.Scheme(s)
	return b
}

// ComponentRegistry sets the component registry.
func (b *OperationBuilder) ComponentRegistry(resolver ctf.ComponentResolver) *OperationBuilder {
	b.Builder.ComponentRegistry(resolver)
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
		ComponentDescriptor:             b.cd,
		BlobResolver:                    b.blobResolver,
		ResolvedComponentDescriptorList: b.resolvedComponentDescriptorList,
	}

	if b.context == nil {
		newCtx, err := GetInstallationContext(ctx, instOp.Client(), instOp.Inst.GetInstallation())
		if err != nil {
			return nil, err
		}
		b.context = newCtx
	}
	instOp.context = *b.context

	if instOp.ComponentDescriptor == nil {
		cdRef := instOp.Context().External.ComponentDescriptorRef()
		if cdRef == nil {
			return instOp, nil
		}
		var err error
		if b.blobResolver == nil {
			instOp.ComponentDescriptor, instOp.BlobResolver, err = instOp.ComponentsRegistry().
				ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
			if err != nil {
				return nil, err
			}
		} else {
			instOp.ComponentDescriptor, err = instOp.ComponentsRegistry().
				Resolve(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
			if err != nil {
				return nil, err
			}
		}
	}
	if instOp.BlobResolver == nil {
		cdRef := instOp.Context().External.ComponentDescriptorRef()
		if cdRef != nil {
			var err error
			_, instOp.BlobResolver, err = instOp.ComponentsRegistry().
				ResolveWithBlobResolver(ctx, cdRef.RepositoryContext, cdRef.ComponentName, cdRef.Version)
			if err != nil {
				return nil, err
			}
		}
	}
	if instOp.ResolvedComponentDescriptorList == nil {
		var err error
		resolvedCD, err := cdutils.ResolveToComponentDescriptorList(ctx, instOp.ComponentsRegistry(), *instOp.ComponentDescriptor, instOp.Context().External.RepositoryContext, instOp.Context().External.Overwriter)
		if err != nil {
			return nil, err
		}
		instOp.ResolvedComponentDescriptorList = &resolvedCD
	}

	return instOp, nil
}
