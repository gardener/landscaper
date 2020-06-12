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
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// InstallationOperation contains all installation operations
type InstallationOperation struct {
	Operation

	Inst    *Installation
	context *Context
}

// NewInstallationOperation creates a new installation operation
func NewInstallationOperation(ctx context.Context, op Operation, inst *Installation) (*InstallationOperation, error) {
	var err error
	instOp := &InstallationOperation{
		Operation: op,
		Inst:      inst,
	}

	instOp.context, err = instOp.DetermineContext(ctx)
	if err != nil {
		return nil, err
	}

	return instOp, nil
}

// Context returns the context of the operated installation
func (o *InstallationOperation) Context() *Context {
	return o.context
}

// GetRootInstallations returns all root installations in the system
func (o *InstallationOperation) GetRootInstallations(ctx context.Context, opts ...client.ListOption) ([]*lsv1alpha1.ComponentInstallation, error) {
	r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)})

	installationList := &lsv1alpha1.ComponentInstallationList{}
	if err := o.Client().List(ctx, installationList, opts...); err != nil {
		return nil, err
	}

	installations := make([]*lsv1alpha1.ComponentInstallation, len(installationList.Items))
	for i, obj := range installationList.Items {
		inst := obj
		installations[i] = &inst
	}
	return installations, nil
}
