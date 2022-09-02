// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

var _ ImportedBase = &TargetExtensionList{}

// TargetExtensionList is the internal representation of a list of targets.
type TargetExtensionList struct {
	targetExtensions []*TargetExtension
	def              *lsv1alpha1.TargetImport
}

// NewTargetExtensionList creates a new internal targetlist instance from a list of raw targets.
func NewTargetExtensionList(targets []lsv1alpha1.Target, def *lsv1alpha1.TargetImport) *TargetExtensionList {
	res := newTargetExtensionListWithSize(len(targets))
	for i := range targets {
		tmp := NewTargetExtension(&targets[i], nil)
		res.targetExtensions[i] = tmp
	}

	res.def = def

	return res
}

// NewTargetExtensionListWithSize creates a new internal targetlist with a given size.
func newTargetExtensionListWithSize(size int) *TargetExtensionList {
	return &TargetExtensionList{
		targetExtensions: make([]*TargetExtension, size),
	}
}

// SetAllSourceType sets the source type for all targets in the list.
func (t *TargetExtensionList) SetAllSourceType(sourceType lsv1alpha1.DataObjectSourceType) *TargetExtensionList {
	for i := range t.targetExtensions {
		t.targetExtensions[i].SetSourceType(sourceType)
	}
	return t
}

// GetData returns the targets as list of internal go maps.
func (t *TargetExtensionList) GetData() ([]interface{}, error) {
	rawTargets := make([]lsv1alpha1.Target, len(t.targetExtensions))
	for i := range t.targetExtensions {
		rawTargets[i] = *t.targetExtensions[i].GetTarget()
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
func (tl TargetExtensionList) Build(tlName string) ([]*lsv1alpha1.Target, error) {
	newTL := make([]*lsv1alpha1.Target, len(tl.targetExtensions))
	for i := 0; i < len(newTL); i++ {
		tar := tl.targetExtensions[i]
		newTarget := &lsv1alpha1.Target{}
		newTarget.Name = lsv1alpha1helper.GenerateDataObjectNameWithIndex(tar.GetMetadata().Context, tar.GetMetadata().Key, i)
		newTarget.Namespace = tar.GetMetadata().Namespace
		if tar.GetTarget() != nil {
			newTarget.Spec = tar.GetTarget().Spec
			for key, val := range tar.GetTarget().Annotations {
				metav1.SetMetaDataAnnotation(&newTarget.ObjectMeta, key, val)
			}
			for key, val := range tar.GetTarget().Labels {
				kutil.SetMetaDataLabel(newTarget, key, val)
			}
		}
		SetMetadataFromObject(newTarget, tar.GetMetadata())
		tar.SetTarget(newTarget)
		newTL[i] = newTarget
	}
	return newTL, nil
}

// Apply applies data and metadata to a existing target (except owner references).
func (tl TargetExtensionList) Apply(raw *lsv1alpha1.Target, index int) error {
	t := tl.targetExtensions[index]
	raw.Name = lsv1alpha1helper.GenerateDataObjectNameWithIndex(t.GetMetadata().Context, t.GetMetadata().Key, index)
	raw.Namespace = t.GetMetadata().Namespace
	raw.Spec = t.GetTarget().Spec
	SetMetadataFromObject(raw, t.GetMetadata())
	return nil
}

// Imported interface

func (tl *TargetExtensionList) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeTargetList
}

func (tl *TargetExtensionList) IsListTypeImport() bool {
	return true
}

func (tl *TargetExtensionList) GetInClusterObject() client.Object {
	return nil
}
func (tl *TargetExtensionList) GetInClusterObjects() []client.Object {
	res := []client.Object{}
	for _, t := range tl.targetExtensions {
		res = append(res, t.GetTarget())
	}
	return res
}

func (tl *TargetExtensionList) ComputeConfigGeneration() string {
	if len(tl.targetExtensions) == 0 {
		return ""
	}

	hashList := make([]string, len(tl.targetExtensions))
	for k, v := range tl.targetExtensions {
		hashList[k] = v.ComputeConfigGeneration()
	}

	hashListJson, err := json.Marshal(hashList)
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal a list of strings during the computation "+
			"of a target list hash; this should never happen"))
	}

	return string(hashListJson)
}

func (tl *TargetExtensionList) GetListItems() []ImportedBase {
	res := make([]ImportedBase, len(tl.targetExtensions))
	for i := range tl.targetExtensions {
		res[i] = tl.targetExtensions[i]
	}
	return res
}

func (tl *TargetExtensionList) GetImportReference() string {
	return ""
}

func (tl *TargetExtensionList) GetImportDefinition() interface{} {
	return tl.def
}

func (tl *TargetExtensionList) GetTargetExtensions() []*TargetExtension {
	return tl.targetExtensions
}
