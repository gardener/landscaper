package dataobjects

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

type TargetMapExtension struct {
	targetExtensions map[string]*TargetExtension

	// def is the import definition in the installation
	def *lsv1alpha1.TargetImport
}

func NewTargetMapExtension(targetMap map[string]lsv1alpha1.Target, def *lsv1alpha1.TargetImport) *TargetMapExtension {
	ext := TargetMapExtension{
		targetExtensions: make(map[string]*TargetExtension),
		def:              def,
	}

	for key := range targetMap {
		ext.targetExtensions[key] = NewTargetExtension(ptr.To(targetMap[key]), nil)
	}

	return &ext
}

func NewTargetMapExtensionFromList(targets *lsv1alpha1.TargetList, def *lsv1alpha1.TargetImport) (*TargetMapExtension, error) {
	ext := TargetMapExtension{
		targetExtensions: make(map[string]*TargetExtension),
		def:              def,
	}

	for i := range targets.Items {
		target := &targets.Items[i]

		id, err := getTargetMapKeyLabel(target)
		if err != nil {
			return nil, err
		}

		ext.targetExtensions[id] = NewTargetExtension(target, nil)
	}

	return &ext, nil
}

// GetData returns the targets as map[string]interface{}.
func (m *TargetMapExtension) GetData() (map[string]interface{}, error) {
	rawTargets := make(map[string]lsv1alpha1.Target, len(m.targetExtensions))
	for targetMapKey := range m.targetExtensions {
		rawTargets[targetMapKey] = *m.targetExtensions[targetMapKey].GetTarget()
	}
	raw, err := json.Marshal(rawTargets)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (m *TargetMapExtension) Build(_ string) (map[string]*lsv1alpha1.Target, error) {
	newTargetMap := make(map[string]*lsv1alpha1.Target)

	for targetMapKey := range m.targetExtensions {
		tar := m.targetExtensions[targetMapKey]

		newTarget := &lsv1alpha1.Target{}
		newTarget.Name = generateTargetNameWithKey(tar.GetMetadata().Context, tar.GetMetadata().Key, targetMapKey)
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
		newTargetMap[targetMapKey] = newTarget
	}

	return newTargetMap, nil
}

// Apply applies data and metadata to an existing target (except owner references).
func (m *TargetMapExtension) Apply(raw *lsv1alpha1.Target, targetMapKey string) error {
	t := m.targetExtensions[targetMapKey]
	raw.Spec = t.GetTarget().Spec
	SetMetadataFromObject(raw, t.GetMetadata())
	return nil
}

func getTargetMapKeyLabel(target *lsv1alpha1.Target) (string, error) {
	id, ok := target.GetLabels()[lsv1alpha1.DataObjectTargetMapKeyLabel]
	if !ok {
		return "", fmt.Errorf("missing label for target map key")
	}
	return id, nil
}

func SetTargetMapKeyLabel(target *lsv1alpha1.Target, targetMapKey string) {
	kutil.SetMetaDataLabel(target, lsv1alpha1.DataObjectTargetMapKeyLabel, targetMapKey)
}

func generateTargetNameWithKey(context string, name string, targetMapKey string) string {
	return lsv1alpha1helper.GenerateDataObjectName(context, fmt.Sprintf("%s[%s]", name, targetMapKey))
}

func (m *TargetMapExtension) GetImportType() lsv1alpha1.ImportType {
	return lsv1alpha1.ImportTypeTargetMap
}

func (m *TargetMapExtension) IsListTypeImport() bool {
	return false
}

func (m *TargetMapExtension) GetInClusterObject() client.Object {
	return nil
}

func (m *TargetMapExtension) GetInClusterObjects() []client.Object {
	res := []client.Object{}
	for _, t := range m.targetExtensions {
		res = append(res, t.GetTarget())
	}
	return res
}

func (m *TargetMapExtension) ComputeConfigGeneration() string {
	if len(m.targetExtensions) == 0 {
		return ""
	}

	hashMap := make(map[string]string)
	for k, v := range m.targetExtensions {
		hashMap[k] = v.ComputeConfigGeneration()
	}

	hashMapJson, err := json.Marshal(hashMap)
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal a map of strings during the computation "+
			"of a target map hash; this should never happen"))
	}

	return string(hashMapJson)
}

func (m *TargetMapExtension) GetListItems() []ImportedBase {
	res := make([]ImportedBase, len(m.targetExtensions))
	i := 0
	for key := range m.targetExtensions {
		res[i] = m.targetExtensions[key]
		i++
	}
	return res
}

func (m *TargetMapExtension) GetImportReference() string {
	return ""
}

func (m *TargetMapExtension) GetImportDefinition() interface{} {
	return m.def
}

func (m *TargetMapExtension) GetTargetExtensions() map[string]*TargetExtension {
	return m.targetExtensions
}
