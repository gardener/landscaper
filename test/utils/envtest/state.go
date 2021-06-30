// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
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
	Generic       map[string]client.Object
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
		Generic:       map[string]client.Object{},
	}
}

// AddsResources to the current state
func (s *State) AddResources(objects ...client.Object) error {
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

type CreateOptions struct {
	// UpdateStatus also updates the status after the objects creation
	UpdateStatus bool
}

// ApplyOptions applies all options from create options to the object
func (o *CreateOptions) ApplyOptions(options ...CreateOption) error {
	for _, obj := range options {
		if err := obj.ApplyOption(o); err != nil {
			return err
		}
	}
	return nil
}

type CreateOption interface {
	ApplyOption(options *CreateOptions) error
}

type UpdateStatus bool

func (s UpdateStatus) ApplyOption(options *CreateOptions) error {
	options.UpdateStatus = bool(s)
	return nil
}

// Create creates or updates a kubernetes resources and adds it to the current state
func (s *State) Create(ctx context.Context, c client.Client, obj client.Object, opts ...CreateOption) error {
	options := &CreateOptions{}
	if err := options.ApplyOptions(opts...); err != nil {
		return err
	}
	if err := c.Create(ctx, obj); err != nil {
		return err
	}

	if options.UpdateStatus {
		if err := c.Status().Update(ctx, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
	}
	return s.AddResources(obj)
}

// InitResources creates a new isolated environment with its own namespace.
func (s *State) InitResources(ctx context.Context, c client.Client, resourcesPath string) error {
	// parse state and create resources in cluster
	resources, err := parseResources(resourcesPath, s)
	if err != nil {
		return err
	}

	for _, obj := range resources {
		if err := s.Create(ctx, c, obj, UpdateStatus(true)); err != nil {
			return err
		}
	}

	return nil
}

// CleanupState cleans up a test environment.
// todo: remove finalizers of all objects in state
func (s *State) CleanupState(ctx context.Context, c client.Client, timeout *time.Duration) error {
	if timeout == nil {
		t := 30 * time.Second
		timeout = &t
	}
	for _, obj := range s.DeployItems {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.Executions {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.Installations {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.DataObjects {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.Targets {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.Secrets {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}
	for _, obj := range s.ConfigMaps {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}

	for _, obj := range s.Generic {
		if err := CleanupForObject(ctx, c, obj, *timeout); err != nil {
			return err
		}
	}

	// also remove all pending pods in the namespace if the container deployer leaves some pods
	pods := &corev1.PodList{}
	if err := c.List(ctx, pods, &client.ListOptions{Namespace: s.Namespace}); err != nil {
		return fmt.Errorf("unable to list pods in %q: %w", s.Namespace, err)
	}
	for _, obj := range pods.Items {
		if err := CleanupForObject(ctx, c, &obj, *timeout); err != nil {
			return err
		}
	}

	ns := &corev1.Namespace{}
	ns.Name = s.Namespace
	return c.Delete(ctx, ns)
}

// CleanupForObject cleans up a object from a cluster
func CleanupForObject(ctx context.Context, c client.Client, obj client.Object, timeout time.Duration) error {
	if err := c.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// try to do a graceful cleanup
	if obj.GetDeletionTimestamp().IsZero() {
		if err := c.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := WaitForObjectToBeDeleted(ctx, c, obj, timeout); err != nil {
		if err := removeFinalizer(ctx, c, obj); err != nil {
			return err
		}
	}
	return nil
}

// WaitForObjectToBeDeleted waits for a object to be deleted.
func WaitForObjectToBeDeleted(ctx context.Context, c client.Client, obj client.Object, timeout time.Duration) error {
	return wait.PollImmediate(2*time.Second, timeout, func() (done bool, err error) {
		uObj := obj.DeepCopyObject().(client.Object)
		if err := c.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, uObj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func removeFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if len(obj.GetFinalizers()) == 0 {
		return nil
	}
	if err := c.Get(ctx, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()), obj); err != nil {
		return err
	}
	currObj := obj.DeepCopyObject().(client.Object)

	obj.SetFinalizers([]string{})
	patch := client.MergeFrom(currObj)
	if err := c.Patch(ctx, obj, patch); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to remove finalizer from object: %w", err)
	}
	return nil
}
