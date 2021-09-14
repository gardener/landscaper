// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
)

// CombinedPhase returns the phase of the referenced execution.
func CombinedPhase(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation) (lsv1alpha1.ExecutionPhase, error) {
	exec, err := GetExecutionForInstallation(ctx, kubeClient, inst)
	if err != nil {
		return "", err
	}
	if exec == nil {
		return "", nil
	}

	if exec.Generation != exec.Status.ObservedGeneration {
		return lsv1alpha1.ExecutionPhaseProgressing, nil
	}

	return exec.Status.Phase, nil
}

// GetExportedValues returns the exported values of the execution
func (o *ExecutionOperation) GetExportedValues(ctx context.Context, inst *installations.Installation) (*dataobjects.DataObject, error) {
	exec := &lsv1alpha1.Execution{}
	if err := o.Client().Get(ctx, kutil.ObjectKey(inst.Info.Name, inst.Info.Namespace), exec); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	doName := lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromExecution(exec), "")
	rawDO := &lsv1alpha1.DataObject{}
	if err := o.Client().Get(ctx, kutil.ObjectKey(doName, o.Inst.Info.Namespace), rawDO); err != nil {
		return nil, err
	}

	return dataobjects.NewFromDataObject(rawDO)
}
