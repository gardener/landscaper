package utils

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// this file contains methods which are only required during upgrade situations and could be removed
// after some time

// added 29th Jan 2024
func CheckIfNewContextDeletion(doList *lsv1alpha1.DataObjectList, targetList *lsv1alpha1.TargetList) bool {

	for i := range doList.Items {
		if kubernetes.HasLabel(&doList.Items[i].ObjectMeta, lsv1alpha1.DataObjectJobIDLabel) {
			return true
		}
	}

	for i := range targetList.Items {
		if kubernetes.HasLabel(&targetList.Items[i].ObjectMeta, lsv1alpha1.DataObjectJobIDLabel) {
			return true
		}
	}

	return false
}
