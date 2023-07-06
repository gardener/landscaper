// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper

import (
	"context"
	"fmt"

	lserror "github.com/gardener/landscaper/apis/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/landscaper/pkg/utils/dependencies"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
)

type ReconcileHelper struct {
	*installations.Operation
	ctx          context.Context
	parent       *installations.InstallationImportsAndBlueprint   // parent installation or nil in case of a root installation
	importStatus *installations.ImportStatus                      // we need to store the 'old' import status, as it is overwritten during import loading
	siblings     map[string]*installations.InstallationAndImports // all installations in the same namespace with the same parent, mapped by their names for faster lookup
	imports      *imports.Imports                                 // struct containing the imports
}

func NewReconcileHelper(ctx context.Context, op *installations.Operation) (*ReconcileHelper, error) {
	rh := &ReconcileHelper{
		ctx:       ctx,
		Operation: op,
	}

	// copy import status
	// This is somewhat ugly, maybe we can somehow refactor the updating of the import status out of the import loading methods?
	rh.importStatus = &installations.ImportStatus{
		Data:   make(map[string]*lsv1alpha1.ImportStatus, len(rh.Inst.ImportStatus().Data)),
		Target: make(map[string]*lsv1alpha1.ImportStatus, len(rh.Inst.ImportStatus().Target)),
	}
	for k, v := range rh.Inst.ImportStatus().Data {
		rh.importStatus.Data[k] = v.DeepCopy()
	}
	for k, v := range rh.Inst.ImportStatus().Target {
		rh.importStatus.Target[k] = v.DeepCopy()
	}

	if err := rh.fetchParent(); err != nil {
		return nil, err
	}

	if err := rh.fetchSiblings(); err != nil {
		return nil, err
	}

	return rh, nil
}

///// VALIDATION METHODS /////

// ImportsUpToDate returns true if the export configGeneration of each import is equal to the configGeneration in the import status
// meaning that the imports have not been updated since they have last been imported.
// It does not check whether the exporting installation is up-to-date.
func (rh *ReconcileHelper) ImportsUpToDate() (bool, error) {
	returnAndSetCondition := func(utd bool) bool {
		cond := lsv1alpha1helper.GetOrInitCondition(rh.Inst.GetInstallation().Status.Conditions, lsv1alpha1.ValidateImportsCondition)
		outdatedImportsReason := "OutdatedImports"
		var condValue lsv1alpha1.ConditionStatus
		var msg string
		if utd {
			condValue = lsv1alpha1.ConditionFalse
			msg = "All imports are up-to-date"
		} else {
			condValue = lsv1alpha1.ConditionTrue
			msg = "At least one import is outdated"
		}
		rh.Inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, condValue, outdatedImportsReason, msg))
		return utd
	}

	imps, err := rh.ImportsSatisfied()
	if err != nil {
		return false, err
	}

	for _, imp := range imps.All() {
		// fetch stored config generation from installation status
		storedConfigGen, storedConfigGens, err := rh.getConfigGenerationsFromImportStatus(imp)
		if err != nil {
			return false, err
		}
		if !imp.IsListTypeImport() {
			// handle non-list-type imports

			owner := imp.GetOwnerReference()
			var configGen string
			if installations.OwnerReferenceIsInstallationButNoParent(owner, rh.Inst.GetInstallation()) {
				// owner is an installation, get configGeneration from its status
				configGen, err = rh.getOwnerGeneration(owner)
				if err != nil {
					return false, err
				}
			} else {
				// owner is not an installation, compute alternative config generation
				configGen = imp.ComputeConfigGeneration()
			}
			if configGen != storedConfigGen {
				// something has changed since last imported
				return returnAndSetCondition(false), nil
			}
		} else {
			// handle list-type imports

			owners := imp.GetOwnerReferences()
			for objectName, owner := range owners {
				var configGen string
				storedConfigGen = ""
				if storedConfigGens != nil {
					storedConfigGen = storedConfigGens[objectName]
				}
				if installations.OwnerReferenceIsInstallationButNoParent(owner, rh.Inst.GetInstallation()) {
					// owner is an installation, get configGeneration from its status
					configGen, err = rh.getOwnerGeneration(owner)
					if err != nil {
						return false, err
					}
				} else {
					// owner is not an installation, compute alternative config generation
					configGen = imp.ComputeConfigGenerationForItem(objectName)
				}
				if configGen != storedConfigGen {
					// something has changed since last imported
					return returnAndSetCondition(false), nil
				}
			}
		}
	}

	return returnAndSetCondition(true), nil
}

//nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
func (rh *ReconcileHelper) GetPredecessors(installation *lsv1alpha1.Installation, predecessorNames sets.String) (map[string]*installations.InstallationAndImports, error) {
	predecessorMap := map[string]*installations.InstallationAndImports{}

	for name := range predecessorNames {
		predecessor := rh.siblings[name]
		if predecessor == nil {
			return nil, fmt.Errorf("internal error: sibling %q is nil", name)
		}

		predecessorMap[name] = predecessor
	}

	return predecessorMap, nil
}

func (rh *ReconcileHelper) AllPredecessorsFinished(installation *lsv1alpha1.Installation,
	predecessorMap map[string]*installations.InstallationAndImports) lserror.LsError {
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

func (rh *ReconcileHelper) AllPredecessorsSucceeded(installation *lsv1alpha1.Installation, predecessorMap map[string]*installations.InstallationAndImports) error {
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
			rh.Operation.Client(), rh.Operation.ComponentsRegistry())
		if err != nil {
			return err
		}
	}
	rh.parent = parent
	return nil
}

// fetchSiblings sets the siblings field
func (rh *ReconcileHelper) fetchSiblings() error {
	rawSiblings := rh.Context().Siblings
	rh.siblings = map[string]*installations.InstallationAndImports{}
	for _, elem := range rawSiblings {
		rh.siblings[elem.GetInstallation().Name] = elem
	}
	return nil
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
func (rh *ReconcileHelper) FetchPredecessors() sets.String {
	inst := rh.Inst.GetInstallation()
	siblingInsts := []*lsv1alpha1.Installation{}
	for _, next := range rh.siblings {
		siblingInsts = append(siblingInsts, next.GetInstallation())
	}

	return dependencies.FetchPredecessorsFromInstallation(inst, siblingInsts)
}

///// AUXILIARY FUNCTIONS /////

// getOwnerGeneration returns the config generation of the owner, if the owner is an installation
func (rh *ReconcileHelper) getOwnerGeneration(owner *metav1.OwnerReference) (string, error) {
	if !installations.OwnerReferenceIsInstallation(owner) {
		// validation only possible for installations
		return "", nil
	}
	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: rh.Inst.GetInstallation().Namespace}

	if lsv1alpha1helper.ReferenceIsObject(ref, rh.Inst.GetInstallation()) {
		return rh.Inst.GetInstallation().Status.ConfigGeneration, nil
	}

	if rh.parent != nil && lsv1alpha1helper.ReferenceIsObject(ref, rh.parent.GetInstallation()) {
		// import comes from the parent
		return rh.parent.GetInstallation().Status.ConfigGeneration, nil
	}

	owningInst, ok := rh.siblings[owner.Name]
	if ok {
		// import is exported by sibling
		return owningInst.GetInstallation().Status.ConfigGeneration, nil
	}

	return "", fmt.Errorf("owner reference %q refers to an installation which is neither the parent nor a sibling", ref.NamespacedName().String())
}

// getConfigGenerationsFromImportStatus gets the config generation(s) for the given import
// If the import is a list-type import, the second argument will contain a mapping from the in-cluster object names to their respective config generations.
// Otherwise, the first argument will contain the config generation.
// The 'unused' return value will be set to its default value.
func (rh *ReconcileHelper) getConfigGenerationsFromImportStatus(imp *dataobjects.Imported) (string, map[string]string, error) {
	var err error
	var importStatus lsv1alpha1.ImportStatus
	var configGen string
	var configGens map[string]string
	switch imp.GetImportType() {
	case lsv1alpha1.ImportTypeData:
		importStatus, err = rh.importStatus.GetData(imp.GetImportName())
		if err == nil {
			configGen = importStatus.ConfigGeneration
		}
	case lsv1alpha1.ImportTypeTarget:
		importStatus, err = rh.importStatus.GetTarget(imp.GetImportName())
		if err == nil {
			configGen = importStatus.ConfigGeneration
		}
	case lsv1alpha1.ImportTypeTargetList:
		importStatus, err = rh.importStatus.GetTarget(imp.GetImportName())
		if err == nil {
			configGens = map[string]string{}
			for _, ts := range importStatus.Targets {
				configGens[ts.Target] = ts.ConfigGeneration
			}
		}
	default:
		return "", nil, fmt.Errorf("unknown import type %q", imp.GetImportName())
	}
	// errors while fetching the import status are ignored
	// as an non-existing import status most probably means that it belongs to a new import which hasn't been imported before
	return configGen, configGens, nil
}
