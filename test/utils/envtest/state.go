// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// State contains the state of initialized fake client
type State struct {
	Namespace     string
	Installations map[string]*lsv1alpha1.Installation
	Executions    map[string]*lsv1alpha1.Execution
	DeployItems   map[string]*lsv1alpha1.DeployItem
	DataObjects   map[string]*lsv1alpha1.DataObject
	Targets       map[string]*lsv1alpha1.Target
	Secrets       map[string]*corev1.Secret
	ConfigMaps    map[string]*corev1.ConfigMap
	Generic       map[string]Object
}

// NewState initializes a new state.
func NewState() *State {
	return &State{
		Installations: make(map[string]*lsv1alpha1.Installation),
		Executions:    make(map[string]*lsv1alpha1.Execution),
		DeployItems:   make(map[string]*lsv1alpha1.DeployItem),
		DataObjects:   make(map[string]*lsv1alpha1.DataObject),
		Targets:       make(map[string]*lsv1alpha1.Target),
		Secrets:       make(map[string]*corev1.Secret),
		ConfigMaps:    make(map[string]*corev1.ConfigMap),
		Generic:       map[string]Object{},
	}
}

// AddsResources to the current state
func (s *State) AddResources(objects ...Object) error {
	for _, obj := range objects {
		switch o := obj.(type) {
		case *lsv1alpha1.Installation:
			s.Installations[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *lsv1alpha1.Execution:
			s.Executions[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *lsv1alpha1.DeployItem:
			s.DeployItems[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *lsv1alpha1.DataObject:
			s.DataObjects[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *lsv1alpha1.Target:
			s.Targets[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *corev1.Secret:
			// add stringdata and data
			if o.Data == nil {
				o.Data = make(map[string][]byte)
			}
			for key, data := range o.StringData {
				o.Data[key] = []byte(data)
			}

			s.Secrets[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		case *corev1.ConfigMap:
			s.ConfigMaps[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o
		default:
			s.Generic[types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}.String()] = o
		}
	}
	return nil
}

// CreateOrUpdate creates or updates a kubernetes resources and adds it to the current state
func (s *State) Create(ctx context.Context, c client.Client, obj Object) error {
	if err := c.Create(ctx, obj); err != nil {
		return err
	}
	if err := c.Status().Update(ctx, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return s.AddResources(obj)
}
