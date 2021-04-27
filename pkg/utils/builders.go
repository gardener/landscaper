// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// DeployItemBuilder is a helper struct to build deploy items
type DeployItemBuilder struct {
	scheme                *runtime.Scheme
	ObjectKey             *lsv1alpha1.ObjectReference
	Type                  string
	ProviderConfiguration runtime.Object
	target                *lsv1alpha1.ObjectReference
	annotations           map[string]string
}

// NewDeployItemBuilder creates a new deploy item builder
func NewDeployItemBuilder(tType string) *DeployItemBuilder {
	return &DeployItemBuilder{
		Type: tType,
	}
}

// DeepCopy creates a deep copy of the builder and its options.
// Note that the scheme is not deep copied.
func (b *DeployItemBuilder) DeepCopy() *DeployItemBuilder {
	newBldr := NewDeployItemBuilder(b.Type).
		Scheme(b.scheme)
	if b.ProviderConfiguration != nil {
		newBldr.ProviderConfig(b.ProviderConfiguration.DeepCopyObject())
	}
	if b.target != nil {
		newBldr.TargetFromObjectRef(b.target.DeepCopy())
	}
	if b.ObjectKey != nil {
		newBldr.Key(b.ObjectKey.Namespace, b.ObjectKey.Name)
	}
	if b.annotations != nil {
		for key, val := range b.annotations {
			newBldr.AddAnnotation(key, val)
		}
	}
	return newBldr
}

// Scheme sets the deployitem scheme that is used for the provider config
func (b *DeployItemBuilder) Scheme(scheme *runtime.Scheme) *DeployItemBuilder {
	b.scheme = scheme
	return b
}

// ProviderConfig sets the deployitem provider configuration.
func (b *DeployItemBuilder) ProviderConfig(obj runtime.Object) *DeployItemBuilder {
	b.ProviderConfiguration = obj
	return b
}

// TargetFromObjectRef sets the deployitem's target.
func (b *DeployItemBuilder) TargetFromObjectRef(tgt *lsv1alpha1.ObjectReference) *DeployItemBuilder {
	b.target = tgt
	return b
}

// TargetFromObjectKey sets the deployitem's target from a client key.
func (b *DeployItemBuilder) TargetFromObjectKey(tgt client.ObjectKey) *DeployItemBuilder {
	b.target = &lsv1alpha1.ObjectReference{
		Name:      tgt.Name,
		Namespace: tgt.Namespace,
	}
	return b
}

// Target sets the deployitem's target from a name and namespace
func (b *DeployItemBuilder) Target(namespace, name string) *DeployItemBuilder {
	b.target = &lsv1alpha1.ObjectReference{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// Key sets the deployitem's namespace and name.
func (b *DeployItemBuilder) Key(namespace, name string) *DeployItemBuilder {
	b.ObjectKey = &lsv1alpha1.ObjectReference{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// AddAnnotation sets a annotation for the deployitem
func (b *DeployItemBuilder) AddAnnotation(key, val string) *DeployItemBuilder {
	if b.annotations == nil {
		b.annotations = map[string]string{}
	}
	b.annotations[key] = val
	return b
}

// Build creates the deploy items using the given options.
func (b *DeployItemBuilder) Build() (*lsv1alpha1.DeployItem, error) {
	b.applyDefaults()
	if err := b.Validate(); err != nil {
		return nil, err
	}

	ext, err := kutil.ConvertToRawExtension(b.ProviderConfiguration, b.scheme)
	if err != nil {
		return nil, err
	}

	di := &lsv1alpha1.DeployItem{}
	di.Spec.Type = lsv1alpha1.DeployItemType(b.Type)
	di.Spec.Target = b.target
	di.Spec.Configuration = ext

	if b.ObjectKey != nil {
		di.Namespace = b.ObjectKey.Namespace
		di.Name = b.ObjectKey.Name
	}
	if b.annotations != nil {
		di.SetAnnotations(b.annotations)
	}
	return di, nil
}

func (b *DeployItemBuilder) applyDefaults() {
	if b.scheme == nil {
		b.scheme = api.Scheme
	}
}

func (b *DeployItemBuilder) Validate() error {
	if len(b.Type) == 0 {
		return errors.New("a type has to be defined")
	}
	if b.ProviderConfiguration == nil {
		return errors.New("a provider configuration has to be defined")
	}
	return nil
}

// TargetBuilder is a helper struct to build targets
type TargetBuilder struct {
	Type          string
	ObjectKey     *lsv1alpha1.ObjectReference
	Configuration interface{}
	annotations   map[string]string
}

// NewTargetBuilder creates a new target builder
func NewTargetBuilder(tType string) *TargetBuilder {
	return &TargetBuilder{
		Type: tType,
	}
}

// DeepCopy creates a deep copy of the builder and its options.
// Note that the scheme is not deep copied.
func (b *TargetBuilder) DeepCopy() *TargetBuilder {
	newBldr := NewTargetBuilder(b.Type).
		Config(b.Configuration)

	if b.ObjectKey != nil {
		newBldr.Key(b.ObjectKey.Namespace, b.ObjectKey.Name)
	}
	if b.annotations != nil {
		for key, val := range b.annotations {
			newBldr.AddAnnotation(key, val)
		}
	}
	return newBldr
}

// Key sets the deployitem's namespace and name.
func (b *TargetBuilder) Key(namespace, name string) *TargetBuilder {
	b.ObjectKey = &lsv1alpha1.ObjectReference{
		Name:      name,
		Namespace: namespace,
	}
	return b
}

// AddAnnotation sets a annotation for the deployitem
func (b *TargetBuilder) AddAnnotation(key, val string) *TargetBuilder {
	if b.annotations == nil {
		b.annotations = map[string]string{}
	}
	b.annotations[key] = val
	return b
}

// Config sets the target config that is used for the provider config
func (b *TargetBuilder) Config(obj interface{}) *TargetBuilder {
	b.Configuration = obj
	return b
}

// Build creates the target using the given options.
func (b *TargetBuilder) Build() (*lsv1alpha1.Target, error) {
	configBytes, err := json.Marshal(b.Configuration)
	if err != nil {
		return nil, fmt.Errorf("unable to decode target config: %w", err)
	}

	target := &lsv1alpha1.Target{}
	target.Spec.Type = lsv1alpha1.TargetType(b.Type)
	target.Spec.Configuration = lsv1alpha1.NewAnyJSON(configBytes)
	if b.ObjectKey != nil {
		target.Namespace = b.ObjectKey.Namespace
		target.Name = b.ObjectKey.Name
	}
	if b.annotations != nil {
		target.SetAnnotations(b.annotations)
	}
	return target, nil
}
