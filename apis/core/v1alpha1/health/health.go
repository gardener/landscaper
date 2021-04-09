// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

var (
	installationConditionTypesTrue = []lsv1alpha1.ConditionType{
		lsv1alpha1.EnsureSubInstallationsCondition,
		lsv1alpha1.ReconcileExecutionCondition,
		lsv1alpha1.CreateImportsCondition,
		lsv1alpha1.CreateExportsCondition,
		lsv1alpha1.EnsureExecutionsCondition,
		lsv1alpha1.ValidateExportCondition,
	}

	installationConditionTypesFalse = []lsv1alpha1.ConditionType{
		lsv1alpha1.ValidateImportsCondition,
	}
)

// CheckInstallation checks if the given Installation is healthy
// An installation is healthy if
// * Its observed generation is up-to-date
// * No annotation landscaper.gardener.cloud/operation is set
// * No lastError is in the status
// * A last operation in state succeeded is present
// * landscaperv1alpha1.ComponentPhaseSucceeded
func CheckInstallation(installation *lsv1alpha1.Installation) error {
	if installation.Status.ObservedGeneration < installation.Generation {
		return fmt.Errorf("observed generation outdated (%d/%d)", installation.Status.ObservedGeneration, installation.Generation)
	}

	if installation.Status.LastError != nil {
		return fmt.Errorf("last errors is set: %s", installation.Status.LastError.Message)
	}

	if installation.Status.Phase != lsv1alpha1.ComponentPhaseSucceeded {
		return fmt.Errorf("installation phase is not suceeded, but %s", installation.Status.Phase)
	}

	if op, ok := installation.Annotations[lsv1alpha1.OperationAnnotation]; ok {
		return fmt.Errorf("landscaper operation %q is not yet picked up by controller", op)
	}

	for _, conditionType := range installationConditionTypesTrue {
		condition := lsv1alpha1helper.GetCondition(installation.Status.Conditions, conditionType)
		if condition == nil {
			// conditions vary based on what is configured in the installation
			continue
		}

		if err := checkConditionState(string(conditionType), string(lsv1alpha1.ConditionTrue), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	for _, conditionType := range installationConditionTypesFalse {
		condition := lsv1alpha1helper.GetCondition(installation.Status.Conditions, conditionType)
		if condition == nil {
			continue
		}

		if err := checkConditionState(string(conditionType), string(lsv1alpha1.ConditionFalse), string(condition.Status), condition.Reason, condition.Message); err != nil {
			return err
		}
	}

	return nil
}

func checkConditionState(conditionType string, expected, actual, reason, message string) error {
	if expected != actual {
		return fmt.Errorf("condition %q has invalid status %s (expected %s) due to %s: %s",
			conditionType, actual, expected, reason, message)
	}
	return nil
}
