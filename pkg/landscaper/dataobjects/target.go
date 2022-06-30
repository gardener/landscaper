// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

var _ ImportedBase = &Target{}

// Target is the internal representation of a target.
type Target struct {
	Raw        *lsv1alpha1.Target
	FieldValue *lsv1alpha1.FieldValueDefinition
	Metadata   Metadata
	Owner      *metav1.OwnerReference
	Def        *lsv1alpha1.TargetImport
}

// NewFromTarget creates a new internal target instance from a raw target.
func NewFromTarget(target *lsv1alpha1.Target) (*Target, error) {
	return &Target{
		Raw:      target,
		Metadata: GetMetadataFromObject(target, target.Spec.Configuration.RawMessage),
		Owner:    kutil.GetOwner(target.ObjectMeta),
	}, nil
}

// GetData returns the target as internal go map.
func (t *Target) GetData() (interface{}, error) {
	raw, err := json.Marshal(t.Raw)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetContext sets the installation context for the given data object.
func (t *Target) SetContext(ctx string) *Target {
	t.Metadata.Context = ctx
	return t
}

// SetNamespace sets the namespace for the given data object.
func (t *Target) SetNamespace(ns string) *Target {
	t.Metadata.Namespace = ns
	return t
}

// SetSourceType sets the context for the given data object.
func (t *Target) SetSourceType(ctx lsv1alpha1.DataObjectSourceType) *Target {
	t.Metadata.SourceType = ctx
	return t
}

// SetSource sets the source for the given data object.
func (t *Target) SetSource(src string) *Target {
	t.Metadata.Source = src
	return t
}

// SetKey sets the key for the given data object.
func (t *Target) SetKey(key string) *Target {
	t.Metadata.Key = key
	return t
}

// SetOwner sets the owner for the given data object.
func (t *Target) SetOwner(own *metav1.OwnerReference) *Target {
	t.Owner = own
	return t
}

// Build creates a new data object based on the given data and metadata.
// Does not set owner references.
func (t *Target) Build() (*lsv1alpha1.Target, error) {
	newTarget := &lsv1alpha1.Target{}
	newTarget.Name = lsv1alpha1helper.GenerateDataObjectName(t.Metadata.Context, t.Metadata.Key)
	newTarget.Namespace = t.Metadata.Namespace
	if t.Raw != nil {
		newTarget.Spec = t.Raw.Spec
		for key, val := range t.Raw.Annotations {
			metav1.SetMetaDataAnnotation(&newTarget.ObjectMeta, key, val)
		}
		for key, val := range t.Raw.Labels {
			kutil.SetMetaDataLabel(newTarget, key, val)
		}
		t.Metadata.Hash = generateHash(t.Raw.Spec.Configuration.RawMessage)
	}

	SetMetadataFromObject(newTarget, t.Metadata)
	t.Raw = newTarget
	return newTarget, nil
}

// Apply applies data and metadata to an existing target (except owner references).
func (t Target) Apply(raw *lsv1alpha1.Target) error {
	raw.Name = lsv1alpha1helper.GenerateDataObjectName(t.Metadata.Context, t.Metadata.Key)
	raw.Namespace = t.Metadata.Namespace
	raw.Spec = t.Raw.Spec
	t.Metadata.Hash = generateHash(t.Raw.Spec.Configuration.RawMessage)
	SetMetadataFromObject(raw, t.Metadata)
	return nil
}

// Imported interface

func (t *Target) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeTarget
}

func (t *Target) IsListTypeImport() bool {
	return false
}

func (t *Target) GetInClusterObject() client.Object {
	return t.Raw
}
func (t *Target) GetInClusterObjects() []client.Object {
	return nil
}

func (t *Target) ComputeConfigGeneration() string {

	return strconv.FormatInt(t.GetInClusterObject().GetGeneration(), 10)
}

func (t *Target) GetListItems() []ImportedBase {
	return nil
}

func (t *Target) GetImportReference() string {
	return t.Def.Target
}

func (t *Target) GetImportDefinition() interface{} {
	return t.Def
}
