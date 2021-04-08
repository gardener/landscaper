// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package targetselector

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Match checks if the given targets matches all selectors.
func Match(target *lsv1alpha1.Target, selectors []lsv1alpha1.TargetSelector) (bool, error) {
	for i, sel := range selectors {
		ok, err := MatchSelector(target, sel)
		if err != nil {
			return false, fmt.Errorf("unable to match selector %d: %w", i, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// MatchSelector checks if the given targets matches the selector.
// It only passes if all configured selector methods match.
func MatchSelector(target *lsv1alpha1.Target, selector lsv1alpha1.TargetSelector) (bool, error) {
	if len(selector.Annotations) != 0 {
		ok, err := MatchAnnotations(target, selector.Annotations)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	if len(selector.Labels) != 0 {
		ok, err := MatchLabels(target, selector.Labels)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// MatchAnnotations matches a targets annotation for configured requirements.
// All requirements must match in order to match the target.
func MatchAnnotations(target *lsv1alpha1.Target, requirements []lsv1alpha1.Requirement) (bool, error) {
	ann := labels.Set(target.GetAnnotations())
	for _, req := range requirements {
		req1, err := labels.NewRequirement(req.Key, req.Operator, req.Values)
		if err != nil {
			return false, err
		}
		if !req1.Matches(ann) {
			return false, nil
		}
	}
	return true, nil
}

// MatchLabels matches a target's labels for configured requirements.
// All requirements must match in order to match the target.
func MatchLabels(target *lsv1alpha1.Target, requirements []lsv1alpha1.Requirement) (bool, error) {
	labelSet := labels.Set(target.GetLabels())
	for _, req := range requirements {
		req1, err := labels.NewRequirement(req.Key, req.Operator, req.Values)
		if err != nil {
			return false, err
		}
		if !req1.Matches(labelSet) {
			return false, nil
		}
	}
	return true, nil
}
