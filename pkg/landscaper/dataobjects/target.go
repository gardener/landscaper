// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

var _ ImportedBase = &TargetExtension{}

// TargetExtension is the internal representation of a target.
type TargetExtension struct {
	target   *lsv1alpha1.Target
	metadata Metadata
	def      *lsv1alpha1.TargetImport
}

// NewTargetExtension creates a new internal target instance from a raw target.
func NewTargetExtension(target *lsv1alpha1.Target, targetImport *lsv1alpha1.TargetImport) *TargetExtension {
	metadata := Metadata{}
	if target != nil {
		metadata = GetMetadataFromObject(target, GetHashableContent(target))
	}
	return &TargetExtension{
		target:   target,
		metadata: metadata,
		def:      targetImport,
	}
}

// GetData returns the target as internal go map.
func (t *TargetExtension) GetData() (interface{}, error) {
	raw, err := json.Marshal(t.target)
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
func (t *TargetExtension) SetContext(ctx string) *TargetExtension {
	t.metadata.Context = ctx
	return t
}

func (t *TargetExtension) SetJobID(jobID string) *TargetExtension {
	t.metadata.JobID = jobID
	return t
}

// SetNamespace sets the namespace for the given data object.
func (t *TargetExtension) SetNamespace(ns string) *TargetExtension {
	t.metadata.Namespace = ns
	return t
}

// SetSourceType sets the context for the given data object.
func (t *TargetExtension) SetSourceType(ctx lsv1alpha1.DataObjectSourceType) *TargetExtension {
	t.metadata.SourceType = ctx
	return t
}

// SetSource sets the source for the given data object.
func (t *TargetExtension) SetSource(src string) *TargetExtension {
	t.metadata.Source = src
	return t
}

// SetKey sets the key for the given data object.
func (t *TargetExtension) SetKey(key string) *TargetExtension {
	t.metadata.Key = key
	return t
}

// SetIndex sets the index (for list-type objects)
func (t *TargetExtension) SetIndex(idx *int) *TargetExtension {
	t.metadata.Index = idx
	return t
}

func (t *TargetExtension) SetTargetMapKey(targetMapKey *string) *TargetExtension {
	t.metadata.TargetMapKey = targetMapKey
	return t
}

// Apply applies data and metadata to an existing target (except owner references).
func (t *TargetExtension) Apply(target *lsv1alpha1.Target) error {
	target.Name = lsv1alpha1helper.GenerateDataObjectName(t.metadata.Context, t.metadata.Key)
	target.Namespace = t.metadata.Namespace
	target.Spec = t.target.Spec
	for key, val := range t.target.Annotations {
		metav1.SetMetaDataAnnotation(&target.ObjectMeta, key, val)
	}
	for key, val := range t.target.Labels {
		kutil.SetMetaDataLabel(target, key, val)
	}
	tmpMetadata := t.metadata
	tmpMetadata.Hash = generateHash(GetHashableContent(t.target))
	SetMetadataFromObject(target, tmpMetadata)
	return nil
}

// GetHashableContent returns the value of the Target based on which its hash can be computed.
// This is either .Spec.Configuration.RawMessage or a json representation of .Spec.SecretRef.
// If neither is set (or the given target is nil), nil is returned.
func GetHashableContent(t *lsv1alpha1.Target) []byte {
	if t == nil {
		return nil
	}
	if t.Spec.Configuration != nil {
		return t.Spec.Configuration.RawMessage
	} else if t.Spec.SecretRef != nil {
		return []byte(fmt.Sprintf(`{"secretRef": {"name": "%s", "key": "%s"}}`, t.Spec.SecretRef.Name, t.Spec.SecretRef.Key))
	}
	return nil
}

// ApplyNameAndNamespace sets name and namespace based on the given metadata.
func (t *TargetExtension) ApplyNameAndNamespace(target *lsv1alpha1.Target) {
	target.Name = lsv1alpha1helper.GenerateDataObjectName(t.metadata.Context, t.metadata.Key)
	target.Namespace = t.metadata.Namespace
}

// Imported interface

func (t *TargetExtension) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeTarget
}

func (t *TargetExtension) IsListTypeImport() bool {
	return false
}

func (t *TargetExtension) GetInClusterObject() client.Object {
	return t.target
}
func (t *TargetExtension) GetInClusterObjects() []client.Object {
	return nil
}

func (t *TargetExtension) ComputeConfigGeneration() string {
	return strconv.FormatInt(t.GetInClusterObject().GetGeneration(), 10)
}

func (t *TargetExtension) GetListItems() []ImportedBase {
	return nil
}

func (t *TargetExtension) GetImportReference() string {
	return t.def.Target
}

func (t *TargetExtension) GetImportDefinition() interface{} {
	return t.def
}

func (t *TargetExtension) GetTarget() *lsv1alpha1.Target {
	return t.target
}

func (t *TargetExtension) SetTarget(target *lsv1alpha1.Target) {
	t.target = target
}

func (t *TargetExtension) GetMetadata() Metadata {
	return t.metadata
}

func (t *TargetExtension) SetMetadata(metadata Metadata) {
	t.metadata = metadata
}
