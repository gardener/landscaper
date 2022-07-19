// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	install "github.com/gardener/landscaper/apis/core/install"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils"
)

// GetCachelessClient is a helper function that returns a client that can be used before the manager is started
func GetCachelessClient(restConfig *rest.Config) (client.Client, error) {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, err
	}
	install.Install(s)

	return client.New(restConfig, client.Options{Scheme: s})
}

// checkForDuplicateExports checks whether there exists an Installation which already exports an export declared by the given installation.
// The first return value contains an error in case of a duplicate export and nil otherwise.
// The second return value is for other errors and will be nil, if the function worked as expected.
// This whole thing is a little bit sketchy with the conversions, maybe we need to refactor this later ...
func checkForDuplicateExports(ctx context.Context, c client.Client, inst *lsv1alpha1.Installation) (error, error) {
	// fetch all installations in the same namespace with the same parent
	var selector client.ListOption
	if parent, ok := inst.Labels[lsv1alpha1.EncompassedByLabel]; ok {
		selector = client.MatchingLabels(map[string]string{
			lsv1alpha1.EncompassedByLabel: parent,
		})
	} else {
		r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
		if err != nil {
			return nil, fmt.Errorf("internal error: unable to build label requirement: %w", err)
		}
		selector = client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)}
	}
	siblingList := &lsv1alpha1.InstallationList{}
	err := c.List(ctx, siblingList, client.InNamespace(inst.Namespace), selector)
	if err != nil {
		return nil, fmt.Errorf("unable to list installations: %w", err)
	}

	err = utils.CheckForDuplicateExports(inst, siblingList.Items)
	return err, nil
}
