// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"

	lserror "github.com/gardener/landscaper/apis/errors"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/landscaper/pkg/utils/dependencies"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
)

type ReconcileHelper struct {
	*installations.Operation
	ctx         context.Context
	parent      *installations.InstallationImportsAndBlueprint   // parent installation or nil in case of a root installation
	siblingsNew map[string]*installations.InstallationAndImports // all installations in the same namespace with the same parent, mapped by their names for faster lookup
	imports     *imports.Imports                                 // struct containing the imports
}

func NewReconcileHelper(ctx context.Context, op *installations.Operation) (*ReconcileHelper, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "NewReconcileHelper")
	defer pm.StopDebug()

	rh := &ReconcileHelper{
		ctx:       ctx,
		Operation: op,
	}

	if err := rh.fetchParent(); err != nil {
		return nil, err
	}

	//if err := rh.fetchSiblings(); err != nil {
	//	return nil, err
	//}

	return rh, nil
}

///// VALIDATION METHODS /////

//nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
func (rh *ReconcileHelper) GetPredecessors(ctx context.Context, predecessorNames sets.String) (map[string]*installations.InstallationAndImports, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "GetPredecessors")
	defer pm.StopDebug()

	predecessorMap := map[string]*installations.InstallationAndImports{}

	siblings, err := rh.getSiblings()
	if err != nil {
		return nil, err
	}

	for name := range predecessorNames {
		predecessor := siblings[name]
		if predecessor == nil {
			return nil, fmt.Errorf("internal error: sibling %q is nil", name)
		}

		predecessorMap[name] = predecessor
	}

	return predecessorMap, nil
}

func (rh *ReconcileHelper) AllPredecessorsFinished(ctx context.Context, installation *lsv1alpha1.Installation,
	predecessorMap map[string]*installations.InstallationAndImports) lserror.LsError {

	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "AllPredecessorsFinished")
	defer pm.StopDebug()

	// iterate over siblings which is depended on (either directly or transitively) and check if they are 'ready'
	for name := range predecessorMap {
		predecessor := predecessorMap[name]
		reason := string(installations.NotCompletedDependents)

		if installations.IsRootInstallation(installation) {
			if lsv1alpha1helper.HasOperation(predecessor.GetInstallation().ObjectMeta, lsv1alpha1.ReconcileOperation) {
				msg := fmt.Sprintf("depending on installation %q which has reconcile annotation",
					kutil.ObjectKeyFromObject(predecessor.GetInstallation()).String())
				return lserror.NewWrappedError(nil, reason, reason, msg, lsv1alpha1.ErrorForInfoOnly)
			}

			if predecessor.GetInstallation().Status.JobID != predecessor.GetInstallation().Status.JobIDFinished {
				msg := fmt.Sprintf("depending on installation %q which not finished current job %q",
					kutil.ObjectKeyFromObject(predecessor.GetInstallation()).String(), installation.Status.JobID)
				return lserror.NewWrappedError(nil, reason, reason, msg, lsv1alpha1.ErrorForInfoOnly)
			}
		} else {
			if installation.Status.JobID != predecessor.GetInstallation().Status.JobIDFinished {
				msg := fmt.Sprintf("depending on installation %q which not finished current job %q",
					kutil.ObjectKeyFromObject(predecessor.GetInstallation()).String(), installation.Status.JobID)
				return lserror.NewWrappedError(nil, reason, reason, msg, lsv1alpha1.ErrorForInfoOnly)
			}
		}
	}

	return nil
}

func (rh *ReconcileHelper) AllPredecessorsSucceeded(ctx context.Context, installation *lsv1alpha1.Installation, predecessorMap map[string]*installations.InstallationAndImports) error {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "AllPredecessorsSucceeded")
	defer pm.StopDebug()

	for name := range predecessorMap {
		predecessor := predecessorMap[name]

		if predecessor.GetInstallation().Status.InstallationPhase != lsv1alpha1.InstallationPhases.Succeeded {
			reason := string(installations.NotCompletedDependents)
			msg := fmt.Sprintf("depending on installation %q which is not succeeded",
				kutil.ObjectKeyFromObject(predecessor.GetInstallation()).String())
			return lserror.NewWrappedError(nil, reason, reason, msg, lsv1alpha1.ErrorForInfoOnly)
		}
	}

	return nil
}

// ImportsSatisfied returns an error if an import of the installation is not satisfied.
func (rh *ReconcileHelper) ImportsSatisfied() (*imports.Imports, error) {
	if rh.imports == nil {
		if err := rh.fetchImports(); err != nil {
			return nil, err
		}
	}

	return rh.imports, nil
}

///// INFORMATION LOADING METHODS /////

// fetchParent sets the parent field
func (rh *ReconcileHelper) fetchParent() error {
	var parent *installations.InstallationImportsAndBlueprint
	if rh.Operation.Context().Parent != nil {
		var err error
		parent, err = installations.CreateInternalInstallationWithContext(rh.ctx, rh.Operation.Context().Parent.GetInstallation(),
			rh.Operation.LsUncachedClient(), rh.Operation.ComponentsRegistry())
		if err != nil {
			return err
		}
	}
	rh.parent = parent
	return nil
}

// fetchSiblings sets the siblings field
func (rh *ReconcileHelper) getSiblings() (map[string]*installations.InstallationAndImports, error) {
	if rh.siblingsNew != nil {
		return rh.siblingsNew, nil
	}

	rawSiblings, err := rh.Context().GetSiblings(rh.ctx, rh.LsUncachedClient())
	if err != nil {
		return nil, err
	}

	rh.siblingsNew = map[string]*installations.InstallationAndImports{}
	for _, elem := range rawSiblings {
		rh.siblingsNew[elem.GetInstallation().Name] = elem
	}
	return rh.siblingsNew, nil
}

// fetchImports fills the imports field
// It requires siblings and parent
func (rh *ReconcileHelper) fetchImports() error {
	var err error
	con := imports.NewConstructor(rh.Operation)
	rh.imports, err = con.LoadImports(rh.ctx)
	if err != nil {
		return err // todo
	}

	return nil
}

//nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
func (rh *ReconcileHelper) FetchPredecessors(ctx context.Context) (sets.String, error) {
	logger, ctx := logging.FromContextOrNew(ctx, nil)
	pm := utils.StartPerformanceMeasurement(&logger, "FetchPredecessors")
	defer pm.StopDebug()

	inst := rh.Inst.GetInstallation()
	siblingInsts := []*lsv1alpha1.Installation{}

	siblings, err := rh.getSiblings()
	if err != nil {
		return nil, err
	}

	for _, next := range siblings {
		siblingInsts = append(siblingInsts, next.GetInstallation())
	}

	return dependencies.FetchPredecessorsFromInstallation(inst, siblingInsts), nil
}
