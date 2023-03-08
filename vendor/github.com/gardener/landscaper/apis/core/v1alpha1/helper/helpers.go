// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

type TimestampAnnotation string

const (
	ReconcileTimestamp = TimestampAnnotation(v1alpha1.ReconcileTimestampAnnotation)
	AbortTimestamp     = TimestampAnnotation(v1alpha1.AbortTimestampAnnotation)
)

// HasOperation checks if the obj has the given operation annotation
func HasOperation(obj metav1.ObjectMeta, op v1alpha1.Operation) bool {
	currentOp, ok := obj.Annotations[v1alpha1.OperationAnnotation]
	if !ok {
		return false
	}

	return v1alpha1.Operation(currentOp) == op
}

func GetOperation(obj metav1.ObjectMeta) string {
	if obj.Annotations == nil {
		return ""
	}
	return obj.Annotations[v1alpha1.OperationAnnotation]
}

// SetOperation sets the given operation annotation on aa object.
func SetOperation(obj *metav1.ObjectMeta, op v1alpha1.Operation) {
	metav1.SetMetaDataAnnotation(obj, v1alpha1.OperationAnnotation, string(op))
}

func GetTimestampAnnotation(obj metav1.ObjectMeta, ta TimestampAnnotation) (time.Time, error) {
	return time.Parse(time.RFC3339, obj.Annotations[string(ta)])
}

// SetTimestampAnnotationNow sets the timeout annotation with the current timestamp.
func SetTimestampAnnotationNow(obj *metav1.ObjectMeta, ta TimestampAnnotation) {
	metav1.SetMetaDataAnnotation(obj, string(ta), time.Now().Format(time.RFC3339))
}

// SetAbortOperationAndTimestamp sets the annotations for a deploy item abort.
func SetAbortOperationAndTimestamp(obj *metav1.ObjectMeta) {
	SetOperation(obj, v1alpha1.AbortOperation)
	SetTimestampAnnotationNow(obj, AbortTimestamp)
}

// RemoveAbortOperationAndTimestamp removes all abort related annotations
func RemoveAbortOperationAndTimestamp(obj *metav1.ObjectMeta) {
	if len(obj.Annotations) == 0 {
		return
	}
	if val, ok := obj.Annotations[v1alpha1.OperationAnnotation]; ok && val == string(v1alpha1.AbortOperation) {
		delete(obj.Annotations, v1alpha1.OperationAnnotation)
	}
	delete(obj.Annotations, string(AbortTimestamp))
}

func Touch(obj *metav1.ObjectMeta) {
	_, ok := obj.Annotations[v1alpha1.TouchAnnotation]
	if ok {
		delete(obj.Annotations, v1alpha1.TouchAnnotation)
	} else {
		metav1.SetMetaDataAnnotation(obj, v1alpha1.TouchAnnotation, "true")
	}
}

// InitCondition initializes a new Condition with an Unknown status.
func InitCondition(conditionType v1alpha1.ConditionType) v1alpha1.Condition {
	return v1alpha1.Condition{
		Type:               conditionType,
		Status:             v1alpha1.ConditionUnknown,
		Reason:             "ConditionInitialized",
		Message:            "The condition has been initialized but its semantic check has not been performed yet.",
		LastTransitionTime: metav1.Now(),
	}
}

// GetCondition returns the condition with the given <conditionType> out of the list of <conditions>.
// In case the required type could not be found, it returns nil.
func GetCondition(conditions []v1alpha1.Condition, conditionType v1alpha1.ConditionType) *v1alpha1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			c := condition
			return &c
		}
	}
	return nil
}

// GetOrInitCondition tries to retrieve the condition with the given condition type from the given conditions.
// If the condition could not be found, it returns an initialized condition of the given type.
func GetOrInitCondition(conditions []v1alpha1.Condition, conditionType v1alpha1.ConditionType) v1alpha1.Condition {
	if condition := GetCondition(conditions, conditionType); condition != nil {
		return *condition
	}
	return InitCondition(conditionType)
}

// UpdatedCondition updates the properties of one specific condition.
func UpdatedCondition(condition v1alpha1.Condition, status v1alpha1.ConditionStatus, reason, message string, codes ...v1alpha1.ErrorCode) v1alpha1.Condition {
	newCondition := v1alpha1.Condition{
		Type:               condition.Type,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: condition.LastTransitionTime,
		LastUpdateTime:     condition.LastUpdateTime,
		Codes:              codes,
	}

	if !reflect.DeepEqual(condition, newCondition) {
		newCondition.LastUpdateTime = metav1.Now()
	}

	if condition.Status != status {
		newCondition.LastTransitionTime = metav1.Now()
	}

	return newCondition
}

// CreateOrUpdateConditions creates or updates a condition in a condition list.
func CreateOrUpdateConditions(conditions []v1alpha1.Condition, condType v1alpha1.ConditionType, status v1alpha1.ConditionStatus, reason, message string, codes ...v1alpha1.ErrorCode) []v1alpha1.Condition {
	for i, foundCondition := range conditions {
		if foundCondition.Type == condType {
			conditions[i] = UpdatedCondition(conditions[i], status, reason, message, codes...)
			return conditions
		}
	}

	return append(conditions, UpdatedCondition(InitCondition(condType), status, reason, message, codes...))
}

// MergeConditions merges the given <oldConditions> with the <newConditions>. Existing conditions are superseded by
// the <newConditions> (depending on the condition type).
func MergeConditions(oldConditions []v1alpha1.Condition, newConditions ...v1alpha1.Condition) []v1alpha1.Condition {
	var (
		out         = make([]v1alpha1.Condition, 0, len(oldConditions))
		typeToIndex = make(map[v1alpha1.ConditionType]int, len(oldConditions))
	)

	for i, condition := range oldConditions {
		out = append(out, condition)
		typeToIndex[condition.Type] = i
	}

	for _, condition := range newConditions {
		if index, ok := typeToIndex[condition.Type]; ok {
			out[index] = condition
			continue
		}
		out = append(out, condition)
	}

	return out
}

// IsConditionStatus returns if all condition states of all conditions are true.
func IsConditionStatus(conditions []v1alpha1.Condition, status v1alpha1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Status != status {
			return false
		}
	}
	return true
}

// ObjectReferenceFromObject creates a object reference from a k8s object
func ObjectReferenceFromObject(obj metav1.Object) v1alpha1.ObjectReference {
	return v1alpha1.ObjectReference{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

// CreateOrUpdateVersionedObjectReferences creates or updates a element in versioned objectReference slice.
func CreateOrUpdateVersionedObjectReferences(refs []v1alpha1.VersionedObjectReference, ref v1alpha1.ObjectReference, gen int64) []v1alpha1.VersionedObjectReference {
	for i, vref := range refs {
		if vref.ObjectReference == ref {
			refs[i] = v1alpha1.VersionedObjectReference{
				ObjectReference:    ref,
				ObservedGeneration: gen,
			}
			return refs
		}
	}
	return append(refs, v1alpha1.VersionedObjectReference{
		ObjectReference:    ref,
		ObservedGeneration: gen,
	})
}

// GetNamedObjectReference returns the object reference with the given name.
func GetNamedObjectReference(objects []v1alpha1.NamedObjectReference, name string) (v1alpha1.NamedObjectReference, bool) {
	for _, ref := range objects {
		if ref.Name == name {
			return ref, true
		}
	}
	return v1alpha1.NamedObjectReference{}, false
}

// ReferenceIsObject checks if the reference describes the given object.
func ReferenceIsObject(ref v1alpha1.ObjectReference, obj metav1.Object) bool {
	return ref.Name == obj.GetName() && ref.Namespace == obj.GetNamespace()
}

// SetVersionedNamedObjectReference sets the versioned object reference with the given name.
func SetVersionedNamedObjectReference(objects []v1alpha1.VersionedNamedObjectReference, obj v1alpha1.VersionedNamedObjectReference) []v1alpha1.VersionedNamedObjectReference {
	for i, ref := range objects {
		if ref.Name == obj.Name {
			objects[i] = obj
			return objects
		}
	}
	return append(objects, obj)
}

// RemoveVersionedNamedObjectReference removes the first versioned object reference with the given name.
func RemoveVersionedNamedObjectReference(objects []v1alpha1.VersionedNamedObjectReference, name string) []v1alpha1.VersionedNamedObjectReference {
	for i, ref := range objects {
		if ref.Name == name {
			return append(objects[:i], objects[i+1:]...)
		}
	}
	return objects
}

// HasIgnoreAnnotation returns true only if the given object
// has the 'landscaper.gardener.cloud/ignore' annotation
// and its value is 'true'.
func HasIgnoreAnnotation(obj metav1.ObjectMeta) bool {
	v, ok := obj.GetAnnotations()[v1alpha1.IgnoreAnnotation]
	return ok && v == "true"
}

// HasDeleteWithoutUninstallAnnotation returns true only if the given object
// has the 'landscaper.gardener.cloud/delete-without-uninstall' annotation
// and its value is 'true'.
func HasDeleteWithoutUninstallAnnotation(obj metav1.ObjectMeta) bool {
	v, ok := obj.GetAnnotations()[v1alpha1.DeleteWithoutUninstallAnnotation]
	return ok && v == "true"
}

// SetDeployItemToFailed sets status.phase of the DeployItem to a failure phase
// If the DeployItem has a DeletionTimestamp, 'DeleteFailed' is used, otherwise it will be set to 'Failed'.
// Afterwards, the set phase is returned.
// Will do nothing and return an empty string if given a nil pointer.
func SetDeployItemToFailed(di *v1alpha1.DeployItem) v1alpha1.DeployItemPhase {
	if di == nil {
		return ""
	}
	if !di.ObjectMeta.DeletionTimestamp.IsZero() {
		di.Status.Phase = v1alpha1.DeployItemPhases.DeleteFailed
	} else {
		di.Status.Phase = v1alpha1.DeployItemPhases.Failed
	}
	return di.Status.Phase
}
