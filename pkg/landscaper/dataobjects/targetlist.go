// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

var _ ImportedBase = &TargetList{}

// TargetList is the internal representation of a list of targets.
type TargetList struct {
	Targets []*Target
	Def     *lsv1alpha1.TargetImport
}

// NewTargetList creates a new internal targetlist.
func NewTargetList() *TargetList {
	return NewTargetListWithSize(0)
}

// NewTargetListWithSize creates a new internal targetlist with a given size.
func NewTargetListWithSize(size int) *TargetList {
	return &TargetList{
		Targets: make([]*Target, size),
	}
}

// SetAllSourceType sets the source type for all targets in the list.
func (t *TargetList) SetAllSourceType(sourceType lsv1alpha1.DataObjectSourceType) *TargetList {
	for i := range t.Targets {
		t.Targets[i].SetSourceType(sourceType)
	}
	return t
}

// NewFromTargetList creates a new internal targetlist instance from a list of raw targets.
func NewFromTargetList(targets []lsv1alpha1.Target) (*TargetList, error) {
	res := NewTargetListWithSize(len(targets))
	for i := range targets {
		tmp, err := NewFromTarget(&targets[i])
		if err != nil {
			return nil, err
		}
		res.Targets[i] = tmp
	}
	return res, nil
}

// GetData returns the targets as list of internal go maps.
func (t *TargetList) GetData() ([]interface{}, error) {
	rawTargets := make([]lsv1alpha1.Target, len(t.Targets))
	for i := range t.Targets {
		rawTargets[i] = *t.Targets[i].Raw
	}
	raw, err := json.Marshal(rawTargets)
	if err != nil {
		return nil, err
	}
	var data []interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Build creates a new data object based on the given data and metadata.
// Does not set owner references.
func (tl TargetList) Build(tlName string) ([]*lsv1alpha1.Target, error) {
	newTL := make([]*lsv1alpha1.Target, len(tl.Targets))
	for i := 0; i < len(newTL); i++ {
		tar := tl.Targets[i]
		newTarget := &lsv1alpha1.Target{}
		newTarget.Name = lsv1alpha1helper.GenerateDataObjectNameWithIndex(tar.Metadata.Context, tar.Metadata.Key, i)
		newTarget.Namespace = tar.Metadata.Namespace
		if tar.Raw != nil {
			newTarget.Spec = tar.Raw.Spec
			for key, val := range tar.Raw.Annotations {
				metav1.SetMetaDataAnnotation(&newTarget.ObjectMeta, key, val)
			}
			for key, val := range tar.Raw.Labels {
				kutil.SetMetaDataLabel(newTarget, key, val)
			}
		}
		SetMetadataFromObject(newTarget, tar.Metadata)
		tar.Raw = newTarget
		newTL[i] = newTarget
	}
	return newTL, nil
}

// Apply applies data and metadata to a existing target (except owner references).
func (tl TargetList) Apply(raw *lsv1alpha1.Target, index int) error {
	t := tl.Targets[index]
	raw.Name = lsv1alpha1helper.GenerateDataObjectNameWithIndex(t.Metadata.Context, t.Metadata.Key, index)
	raw.Namespace = t.Metadata.Namespace
	raw.Spec = t.Raw.Spec
	SetMetadataFromObject(raw, t.Metadata)
	return nil
}

// Imported interface

func (tl *TargetList) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeTargetList
}

func (tl *TargetList) IsListTypeImport() bool {
	return true
}

func (tl *TargetList) GetInClusterObject() client.Object {
	return nil
}
func (tl *TargetList) GetInClusterObjects() []client.Object {
	res := []client.Object{}
	for _, t := range tl.Targets {
		res = append(res, t.Raw)
	}
	return res
}

func (tl *TargetList) ComputeConfigGeneration() string {
	return ""
}

func (tl *TargetList) GetListItems() []ImportedBase {
	res := make([]ImportedBase, len(tl.Targets))
	for i := range tl.Targets {
		res[i] = tl.Targets[i]
	}
	return res
}

func (tl *TargetList) GetImportReference() string {
	return ""
}

func (tl *TargetList) GetImportDefinition() interface{} {
	return tl.Def
}
