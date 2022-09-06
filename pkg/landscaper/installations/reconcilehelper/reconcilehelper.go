// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/imports"
	"github.com/gardener/landscaper/pkg/landscaper/installations/subinstallations"
)

type ReconcileHelper struct {
	*installations.Operation
	ctx          context.Context
	parent       *installations.Installation                // parent installation or nil in case of a root installation
	importStatus *installations.ImportStatus                // we need to store the 'old' import status, as it is overwritten during import loading
	siblings     map[string]*installations.InstallationBase // all installations in the same namespace with the same parent, mapped by their names for faster lookup
	imports      *imports.Imports                           // struct containing the imports
}

func NewReconcileHelper(ctx context.Context, op *installations.Operation) (*ReconcileHelper, error) {
	rh := &ReconcileHelper{
		ctx:       ctx,
		Operation: op,
	}

	// copy import status
	// This is somewhat ugly, maybe we can somehow refactor the updating of the import status out of the import loading methods?
	rh.importStatus = &installations.ImportStatus{
		Data:                make(map[string]*lsv1alpha1.ImportStatus, len(rh.Inst.ImportStatus().Data)),
		Target:              make(map[string]*lsv1alpha1.ImportStatus, len(rh.Inst.ImportStatus().Target)),
		ComponentDescriptor: make(map[string]*lsv1alpha1.ImportStatus, len(rh.Inst.ImportStatus().ComponentDescriptor)),
	}
	for k, v := range rh.Inst.ImportStatus().Data {
		rh.importStatus.Data[k] = v.DeepCopy()
	}
	for k, v := range rh.Inst.ImportStatus().Target {
		rh.importStatus.Target[k] = v.DeepCopy()
	}
	for k, v := range rh.Inst.ImportStatus().ComponentDescriptor {
		rh.importStatus.ComponentDescriptor[k] = v.DeepCopy()
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

// UpdateAllowed returns an error if the installation cannot be updated due to dependencies or unfulfilled imports.
func (rh *ReconcileHelper) UpdateAllowed(dependedOnSiblings sets.String) error {
	err := rh.InstallationsDependingOnReady(dependedOnSiblings)
	if err != nil {
		// at least one of the installations the current one depends on is not succeeded or has pending changes
		return installations.NewNotCompletedDependentsErrorf(err, "not all installations which are depended on are ready: %s", err.Error())
	}
	err = rh.ImportsSatisfied()
	return err
}

// ImportsUpToDate returns true if the export configGeneration of each import is equal to the configGeneration in the import status
// meaning that the imports have not been updated since they have last been imported.
// It does not check whether the exporting installation is up-to-date.
func (rh *ReconcileHelper) ImportsUpToDate() (bool, error) {
	returnAndSetCondition := func(utd bool) bool {
		cond := lsv1alpha1helper.GetOrInitCondition(rh.Inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)
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

	imps, err := rh.GetImports()
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
			if installations.OwnerReferenceIsInstallationButNoParent(owner, rh.Inst.Info) {
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
				if installations.OwnerReferenceIsInstallationButNoParent(owner, rh.Inst.Info) {
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

func (rh *ReconcileHelper) GetPredecessors(installation *lsv1alpha1.Installation, predecessorNames sets.String) (map[string]*installations.InstallationBase, error) {
	predecessorMap := map[string]*installations.InstallationBase{}

	for name := range predecessorNames {
		predecessor := rh.siblings[name]
		if predecessor == nil {
			return nil, fmt.Errorf("internal error: sibling %q is nil", name)
		}

		predecessorMap[name] = predecessor
	}

	return predecessorMap, nil
}

func (rh *ReconcileHelper) AllPredecessorsFinished(installation *lsv1alpha1.Installation, predecessorMap map[string]*installations.InstallationBase) error {
	// iterate over siblings which is depended on (either directly or transitively) and check if they are 'ready'
	for name := range predecessorMap {
		predecessor := predecessorMap[name]
		if installations.IsRootInstallation(installation) {
			if lsv1alpha1helper.HasOperation(predecessor.Info.ObjectMeta, lsv1alpha1.ReconcileOperation) {
				return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which has reconcile annotation",
					kutil.ObjectKeyFromObject(predecessor.Info).String())
			}

			if predecessor.Info.Status.JobID != predecessor.Info.Status.JobIDFinished {
				return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which not finished current job %q",
					kutil.ObjectKeyFromObject(predecessor.Info).String(), installation.Status.JobID)
			}
		} else {
			if installation.Status.JobID != predecessor.Info.Status.JobIDFinished {
				return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which not finished current job %q",
					kutil.ObjectKeyFromObject(predecessor.Info).String(), installation.Status.JobID)
			}
		}
	}

	return nil
}

func (rh *ReconcileHelper) AllPredecessorsSucceeded(installation *lsv1alpha1.Installation, predecessorMap map[string]*installations.InstallationBase) error {
	for name := range predecessorMap {
		predecessor := predecessorMap[name]

		if predecessor.Info.Status.InstallationPhase != lsv1alpha1.InstallationPhaseSucceeded {
			return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which is not succeeded",
				kutil.ObjectKeyFromObject(predecessor.Info).String())
		}
	}

	return nil
}

// InstallationsDependingOnReady returns nil if all installations the current one depends on are
// - in phase 'Succeeded'
// - up-to-date (observedGeneration == generation)
// - not queued for reconciliation (no 'landscaper.gardener.cloud/operation' annotation with value 'reconcile' or 'forceReconcile')
// Returns an error if any of these requirements is not fulfilled.
// Additionally, an error is returned if the installation has a parent and it is not progressing.
func (rh *ReconcileHelper) InstallationsDependingOnReady(dependedOnSiblings sets.String) error {
	if rh.parent != nil && !lsv1alpha1helper.IsProgressingInstallationPhase(rh.parent.Info.Status.Phase) {
		return installations.NewNotCompletedDependentsErrorf(nil, "parent installation %q is not progressing",
			kutil.ObjectKeyFromObject(rh.parent.Info).String())
	}

	// iterate over siblings which is depended on (either directly or transitively) and check if they are 'ready'
	for dep := range dependedOnSiblings {
		inst := rh.siblings[dep]
		if inst == nil {
			return fmt.Errorf("internal error: sibling %q is nil", dep)
		}

		if inst.Info.Status.Phase != lsv1alpha1.ComponentPhaseSucceeded {
			return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which is not succeeded", kutil.ObjectKeyFromObject(inst.Info).String())
		}

		if inst.Info.Generation != inst.Info.Status.ObservedGeneration {
			return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which is not up-to-date", kutil.ObjectKeyFromObject(inst.Info).String())
		}

		if lsv1alpha1helper.HasOperation(inst.Info.ObjectMeta, lsv1alpha1.ReconcileOperation) || lsv1alpha1helper.HasOperation(inst.Info.ObjectMeta, lsv1alpha1.ForceReconcileOperation) {
			return installations.NewNotCompletedDependentsErrorf(nil, "depending on installation %q which has (force-)reconcile annotation", kutil.ObjectKeyFromObject(inst.Info).String())
		}
	}

	return nil
}

// ImportsSatisfied returns an error if an import of the installation is not satisfied.
// It verifies that all imports
// - exist (indirectly done by the import requirement)
// - are actually exported by the parent or a sibling
func (rh *ReconcileHelper) ImportsSatisfied() error {
	fldPath := field.NewPath("spec", "imports")

	imps, err := rh.GetImports()
	if err != nil {
		return err
	}

	// check data imports
	for _, imp := range rh.Inst.Info.Spec.Imports.Data {
		data, ok := imps.DataObjects[imp.Name]
		impPath := fldPath.Child("data", imp.Name)
		if !ok {
			return installations.NewImportNotSatisfiedErrorf(nil, "%s: import not satisfied", impPath.String())
		}
		if err := rh.checkStateForImport(impPath, *dataobjects.NewImported(imp.Name, data)); err != nil {
			return err
		}
	}

	// check target imports
	for _, imp := range rh.Inst.Info.Spec.Imports.Targets {
		impPath := fldPath.Child("targets", imp.Name)
		// distinguish between single target and targetlist imports
		if len(imp.Target) != 0 {
			// single target import
			target, ok := imps.Targets[imp.Name]
			if !ok {
				return installations.NewImportNotSatisfiedErrorf(nil, "%s: import not satisfied", impPath.String())
			}
			if err := rh.checkStateForImport(impPath, *dataobjects.NewImported(imp.Name, target)); err != nil {
				return err
			}
			continue
		}
		// import has to be a targetlist import
		targets, ok := imps.TargetLists[imp.Name]
		if !ok {
			return installations.NewImportNotSatisfiedErrorf(nil, "%s: import not satisfied", impPath.String())
		}
		if len(imp.TargetListReference) != 0 {
			// targetlist reference to parent's targetlist import
			if err := rh.checkStateForParentImport(impPath, imp.TargetListReference); err != nil {
				return err
			}
			continue
		}
		if imp.Targets != nil {
			// targetlist import consisting of multiple target references
			if len(imp.Targets) != len(targets.GetTargetExtensions()) {
				return installations.NewImportNotSatisfiedErrorf(nil, "%s: targetlist import has wrong number of targets: expected %d, got %d", impPath, len(imp.Targets),
					len(targets.GetTargetExtensions()))
			}
			for _, target := range targets.GetTargetExtensions() {
				if err := rh.checkStateForImport(impPath, *dataobjects.NewImported("", target)); err != nil {
					return err
				}
			}
		}
	}

	// check component descriptor imports
	for _, imp := range rh.Inst.Info.Spec.Imports.ComponentDescriptors {
		impPath := fldPath.Child("componentDescriptors", imp.Name)
		if len(imp.DataRef) != 0 {
			if err := rh.checkStateForParentImport(impPath, imp.DataRef); err != nil {
				return err
			}
		}
		// we can only verify component descriptor list imports which reference a parent import
	}

	return nil
}

// GetImports returns the imports of the installation.
// They are fetched from the cluster if that has not happened before.
func (rh *ReconcileHelper) GetImports() (*imports.Imports, error) {
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
	var parent *installations.Installation
	if rh.Operation.Context().Parent != nil {
		var err error
		parent, err = installations.CreateInternalInstallationWithContext(rh.ctx, rh.Operation.Context().Parent.Info, rh.Operation.Client(), rh.Operation.ComponentsRegistry(), rh.Operation.Overwriter)
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
	rh.siblings = map[string]*installations.InstallationBase{}
	for _, elem := range rawSiblings {
		rh.siblings[elem.Info.Name] = elem
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

// fetchDependencies return set of sibling installation names which this installation depends on, including transitive dependencies
// It requires siblings
func (rh *ReconcileHelper) FetchDependencies() (sets.String, error) {
	dependedOnSiblings := sets.NewString()

	// build helper struct to re-use function from subinstallation template depencency computation
	insts := []*subinstallations.PseudoInstallation{}
	// add current installation
	insts = append(insts, subinstallations.AbstractInstallation(rh.Inst.Info))
	// add siblings
	for _, sib := range rh.siblings {
		insts = append(insts, subinstallations.AbstractInstallation(sib.Info))
	}

	// compute dependencies
	deps, _ := subinstallations.ComputeInstallationDependencies(insts)
	queue := []string{rh.Inst.Info.Name}
	for len(queue) > 0 {
		// pop first element from queue
		elem := queue[0]
		queue = queue[1:]

		// fetch dependencies for element
		curDeps, ok := deps[elem]
		if !ok {
			// should not happen
			return dependedOnSiblings, fmt.Errorf("internal error: installation %q not found in dependency graph", elem)
		}
		for d := range curDeps {
			if !dependedOnSiblings.Has(d) {
				// add sibling to list of depended on siblings
				dependedOnSiblings.Insert(d)
				// queue sibling to fetch transitive dependencies
				queue = append(queue, d)
			}
		}
	}

	return dependedOnSiblings, nil
}

///// AUXILIARY FUNCTIONS /////

// getOwnerGeneration returns the config generation of the owner, if the owner is an installation
func (rh *ReconcileHelper) getOwnerGeneration(owner *metav1.OwnerReference) (string, error) {
	if !installations.OwnerReferenceIsInstallation(owner) {
		// validation only possible for installations
		return "", nil
	}
	ref := lsv1alpha1.ObjectReference{Name: owner.Name, Namespace: rh.Inst.Info.Namespace}

	if lsv1alpha1helper.ReferenceIsObject(ref, rh.Inst.Info) {
		return rh.Inst.Info.Status.ConfigGeneration, nil
	}

	if rh.parent != nil && lsv1alpha1helper.ReferenceIsObject(ref, rh.parent.Info) {
		// import comes from the parent
		return rh.parent.Info.Status.ConfigGeneration, nil
	}

	owningInst, ok := rh.siblings[owner.Name]
	if ok {
		// import is exported by sibling
		return owningInst.Info.Status.ConfigGeneration, nil
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
	case lsv1alpha1.ImportTypeComponentDescriptor:
		// there is no config generation for component descriptor imports
	case lsv1alpha1.ImportTypeComponentDescriptorList:
		// there is no config generation for component descriptor imports
	default:
		return "", nil, fmt.Errorf("unknown import type %q", imp.GetImportName())
	}
	// errors while fetching the import status are ignored
	// as an non-existing import status most probably means that it belongs to a new import which hasn't been imported before
	return configGen, configGens, nil
}

// checkStateForImport verifies that the given import is valid by performing some checks on the owner of the imported object:
// - if the import comes from the parent, it is verified that the parent is 'Progressing' and actually imports the import
// - if the import is owned by an installation which is neither the current one nor its parent, it is verified that
//   the exporting installation is actually exporting this value and is a sibling of the current installation.
func (rh *ReconcileHelper) checkStateForImport(impPath *field.Path, imp dataobjects.Imported) error {
	owner := imp.GetOwnerReference()
	if !installations.OwnerReferenceIsInstallation(owner) {
		// we cannot validate if there is no owner or the owner is not an installation
		return nil
	}
	if owner.Name == rh.Inst.Info.Name {
		// nothing to do if the import is owned by the importing installation
		return nil
	}
	if rh.parent != nil && owner.Name == rh.parent.Info.Name {
		// import comes from parent, verify that parent actually imports it
		if err := rh.checkStateForParentImport(impPath, imp.GetImportReference()); err != nil {
			return err
		}
	} else {
		// import has to come from a sibling
		if err := rh.checkStateForSiblingExport(impPath, owner, imp.GetImportReference(), imp.GetImportType()); err != nil {
			return err
		}
	}
	return nil
}

// checkStateForParentImport returns an error if
// - the given import is not imported by the parent
// - the parent installation is not in a progressing phase
func (rh *ReconcileHelper) checkStateForParentImport(fldPath *field.Path, importName string) error {
	// check if the parent also imports my import
	_, err := rh.parent.GetImportDefinition(importName)
	if err != nil {
		return installations.NewImportNotFoundErrorf(err, "%s: import in parent not found", fldPath.String())
	}

	if utils.IsNewReconcile() {
		return nil
	}

	// parent has to be progressing
	if !lsv1alpha1helper.IsProgressingInstallationPhase(rh.parent.Info.Status.Phase) {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: Parent has to be progressing to get imports", fldPath.String())
	}
	return nil
}

// checkStateForSiblingExport returns an error if
// - the given object reference doesn't belong to a sibling
// - the sibling is not exporting the given value
func (rh *ReconcileHelper) checkStateForSiblingExport(fldPath *field.Path, siblingRef *metav1.OwnerReference, importRef string, importType lsv1alpha1.ImportType) error {
	if siblingRef == nil {
		return nil
	}
	sib, ok := rh.siblings[siblingRef.Name]
	if !ok {
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: installation %s is not a sibling", fldPath.String(), siblingRef.Name)
	}
	if len(importRef) == 0 {
		// import comes from a sibling export, but has no import reference value
		// this should not happen
		return installations.NewImportNotSatisfiedErrorf(nil, "%s: internal error: no import reference for sibling import", fldPath.String())
	}

	// search in the sibling for the export mapping where importmap.from == exportmap.to
	isExporting := false
	switch importType {
	case lsv1alpha1.ImportTypeData:
		for _, def := range sib.Info.Spec.Exports.Data {
			if def.DataRef == importRef {
				isExporting = true
				break
			}
		}
	case lsv1alpha1.ImportTypeTarget, lsv1alpha1.ImportTypeTargetList:
		for _, def := range sib.Info.Spec.Exports.Targets {
			if def.Target == importRef {
				isExporting = true
				break
			}
		}
	case lsv1alpha1.ImportTypeComponentDescriptor, lsv1alpha1.ImportTypeComponentDescriptorList:
		// component descriptors can currently not be exported, so this should never happen
		return installations.NewImportNotFoundErrorf(nil, "%s: component descriptors cannot be imported from siblings", fldPath.String())
	default:
		return fmt.Errorf("%s: unknown import type %q", fldPath.String(), string(importType))
	}
	if !isExporting {
		return installations.NewImportNotFoundErrorf(nil, "%s: export in sibling not found", fldPath.String())
	}

	return nil
}
