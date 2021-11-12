// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package envtest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// State contains the state of initialized fake client
type State struct {
	mux           sync.Mutex
	Client        client.Client
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
		Generic:       make(map[string]client.Object),
	}
}

// NewStateWithClient initializes a new state with a client.
func NewStateWithClient(kubeClient client.Client) *State {
	s := NewState()
	s.Client = kubeClient
	return s
}

// HasClient returns whether a client is configured or not
func (s *State) HasClient() bool {
	return s.Client != nil
}

// AddResources to the current state
func (s *State) AddResources(objects ...client.Object) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, obj := range objects {
		switch o := obj.(type) {
		case *lsv1alpha1.Installation:
			s.Installations[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *lsv1alpha1.Execution:
			s.Executions[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *lsv1alpha1.DeployItem:
			s.DeployItems[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *lsv1alpha1.DataObject:
			s.DataObjects[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *lsv1alpha1.Target:
			s.Targets[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *corev1.Secret:
			// add stringdata and data
			if o.Data == nil {
				o.Data = make(map[string][]byte)
			}
			for key, data := range o.StringData {
				o.Data[key] = []byte(data)
			}

			s.Secrets[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		case *corev1.ConfigMap:
			s.ConfigMaps[types.NamespacedName{Name: o.Name, Namespace: o.Namespace}.String()] = o.DeepCopy()
		default:
			s.Generic[types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}.String()] = o.DeepCopyObject().(client.Object)
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

// CreateWithClient creates or updates a kubernetes resources and adds it to the current state
func (s *State) CreateWithClient(ctx context.Context, c client.Client, obj client.Object, opts ...CreateOption) error {
	options := &CreateOptions{}
	if err := options.ApplyOptions(opts...); err != nil {
		return err
	}
	tmp := obj.DeepCopyObject().(client.Object)
	if err := c.Create(ctx, obj); err != nil {
		return err
	}

	tmp.SetName(obj.GetName())
	tmp.SetNamespace(obj.GetNamespace())
	tmp.SetResourceVersion(obj.GetResourceVersion())
	tmp.SetGeneration(obj.GetGeneration())
	tmp.SetUID(obj.GetUID())
	tmp.SetCreationTimestamp(obj.GetCreationTimestamp())
	if options.UpdateStatus {
		if err := c.Status().Update(ctx, tmp); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
	}
	return s.AddResources(tmp)
}

// Create creates or updates a kubernetes resources and adds it to the current state
func (s *State) Create(ctx context.Context, obj client.Object, opts ...CreateOption) error {
	return s.CreateWithClient(ctx, s.Client, obj, opts...)
}

// InitResourcesWithClient creates a new isolated environment with its own namespace.
func (s *State) InitResourcesWithClient(ctx context.Context, c client.Client, resourcesPath string) error {
	// parse state and create resources in cluster
	resources, err := parseResources(resourcesPath, s)
	if err != nil {
		return err
	}

	resourcesChan := make(chan client.Object, len(resources))

	for _, obj := range resources {
		select {
		case resourcesChan <- obj:
		default:
		}
	}

	injectOwnerUUIDs := func(obj client.Object) error {
		refs := obj.GetOwnerReferences()
		for i, ownerRef := range obj.GetOwnerReferences() {
			uObj := &unstructured.Unstructured{}
			uObj.SetAPIVersion(ownerRef.APIVersion)
			uObj.SetKind(ownerRef.Kind)
			uObj.SetName(ownerRef.Name)
			uObj.SetNamespace(obj.GetNamespace())
			if err := c.Get(ctx, kutil.ObjectKeyFromObject(uObj), uObj); err != nil {
				return fmt.Errorf("no owner found for %s\n", kutil.ObjectKeyFromObject(obj).String())
			}
			refs[i].UID = uObj.GetUID()
		}
		obj.SetOwnerReferences(refs)
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	for obj := range resourcesChan {
		if ctx.Err() != nil {
			return fmt.Errorf("context canceled; check resources as there might be a cyclic dependency")
		}
		objName := kutil.ObjectKeyFromObject(obj).String()
		// create namespaces if not exist before
		if len(obj.GetNamespace()) != 0 {
			ns := &corev1.Namespace{}
			ns.Name = obj.GetNamespace()
			if _, err := controllerutil.CreateOrUpdate(ctx, c, ns, func() error {
				return nil
			}); err != nil {
				return err
			}
		}
		// inject real uuids if possible
		if len(obj.GetOwnerReferences()) != 0 {
			if err := injectOwnerUUIDs(obj); err != nil {
				// try to requeue
				// todo: somehow detect cyclic dependencies (maybe just use a context with an timeout)
				resourcesChan <- obj
				continue
			}
		}
		if err := s.CreateWithClient(ctx, c, obj, UpdateStatus(true)); err != nil {
			return fmt.Errorf("unable to create %s %s: %w", objName, obj.GetObjectKind().GroupVersionKind().String(), err)
		}
		if len(resourcesChan) == 0 {
			close(resourcesChan)
		}
	}

	return nil
}

// InitResources creates a new isolated environment with its own namespace.
func (s *State) InitResources(ctx context.Context, resourcesPath string) error {
	return s.InitResourcesWithClient(ctx, s.Client, resourcesPath)
}

type CleanupOptions struct {
	// Timeout defines the timout to wait the cleanup of an object.
	Timeout *time.Duration
	// WaitForDeletion waits until all resources all successfully deleted.
	WaitForDeletion bool
	// RestConfig specify the rest config which is used to remove the namespace.
	RestConfig *rest.Config
}

// ApplyOptions applies all options from create options to the object
func (o *CleanupOptions) ApplyOptions(options ...CleanupOption) error {
	for _, obj := range options {
		if err := obj.ApplyOption(o); err != nil {
			return err
		}
	}
	return nil
}

type CleanupOption interface {
	ApplyOption(options *CleanupOptions) error
}

// WithCleanupTimeout configures the cleanup timeout
type WithCleanupTimeout time.Duration

func (s WithCleanupTimeout) ApplyOption(options *CleanupOptions) error {
	t := time.Duration(s)
	options.Timeout = &t
	return nil
}

// WaitForDeletion configures the cleanup to wait for all resources to be deleted.
type WaitForDeletion bool

func (s WaitForDeletion) ApplyOption(options *CleanupOptions) error {
	options.WaitForDeletion = bool(s)
	return nil
}

// WithRestConfig configures the rest config
func WithRestConfig(cfg *rest.Config) WithRestConfigOption {
	return WithRestConfigOption{
		RestConfig: cfg,
	}
}

// WithRestConfigOption configures the rest config
type WithRestConfigOption struct {
	RestConfig *rest.Config
}

func (s WithRestConfigOption) ApplyOption(options *CleanupOptions) error {
	options.RestConfig = s.RestConfig
	return nil
}

// CleanupStateWithClient cleans up a test environment.
// todo: remove finalizers of all objects in state
func (s *State) CleanupStateWithClient(ctx context.Context, c client.Client, opts ...CleanupOption) error {
	options := &CleanupOptions{}
	if err := options.ApplyOptions(opts...); err != nil {
		return err
	}
	timeout := options.Timeout
	if timeout == nil {
		t := 30 * time.Second
		timeout = &t
	}

	s.mux.Lock()
	defer s.mux.Unlock()
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
	if err := c.Delete(ctx, ns); err != nil {
		return err
	}
	// the ns will never get removed as there is no kcm to clean it up.
	// So we simply delete it.
	if options.RestConfig != nil {
		if err := removeFinalizerFromNamespace(ctx, options.RestConfig, ns); err != nil {
			return err
		}
		if options.WaitForDeletion {
			return WaitForObjectToBeDeleted(ctx, c, ns, *timeout)
		}
	}

	return nil
}

// CleanupState cleans up a test environment.
// todo: remove finalizers of all objects in state
func (s *State) CleanupState(ctx context.Context, opts ...CleanupOption) error {
	return s.CleanupStateWithClient(ctx, s.Client, opts...)
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
	var (
		lastErr error
		uObj    client.Object
	)
	err := wait.PollImmediate(2*time.Second, timeout, func() (done bool, err error) {
		uObj = obj.DeepCopyObject().(client.Object)
		if err := c.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, uObj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			lastErr = err
			return false, nil
		}
		return false, nil
	})
	if err != nil {
		if lastErr != nil {
			return lastErr
		}
		// try to print the whole object to debug
		d, err2 := json.Marshal(uObj)
		if err2 != nil {
			return err
		}
		return fmt.Errorf("deletion timeout: %s", string(d))
	}
	return nil
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

func removeFinalizerFromNamespace(ctx context.Context, restConfig *rest.Config, ns *corev1.Namespace) error {
	kClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	ns.SetFinalizers([]string{})
	if _, err := kClient.CoreV1().Namespaces().Finalize(ctx, ns, v1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
