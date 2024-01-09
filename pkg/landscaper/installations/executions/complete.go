// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"context"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
)

// GetExportedValues returns the exported values of the execution
func (o *ExecutionOperation) GetExportedValues(ctx context.Context, inst *installations.InstallationImportsAndBlueprint) (*dataobjects.DataObject, error) {
	exec := &lsv1alpha1.Execution{}
	if err := read_write_layer.GetExecution(ctx, o.GetUncacheLsClient(), kutil.ObjectKey(inst.GetInstallation().Name, inst.GetInstallation().Namespace),
		exec, read_write_layer.R000022); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	doName := lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromExecution(exec), "")
	rawDO := &lsv1alpha1.DataObject{}
	if err := o.GetUncacheLsClient().Get(ctx, kutil.ObjectKey(doName, o.Inst.GetInstallation().Namespace), rawDO); err != nil {
		return nil, err
	}

	return dataobjects.NewFromDataObject(rawDO)
}
