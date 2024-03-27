// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lserrors "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	lsutil "github.com/gardener/landscaper/pkg/utils"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Operation contains all installation operations and implements the Operation interface.
type Operation struct {
	*lsoperation.Operation

	Inst                            *InstallationImportsAndBlueprint
	ComponentVersion                model.ComponentVersion
	ResolvedComponentDescriptorList *model.ComponentVersionList
	context                         Scope

	targetLists map[string]*dataobjects.TargetExtensionList
	targetMaps  map[string]*dataobjects.TargetMapExtension
	targets     map[string]*dataobjects.TargetExtension

	// CurrentOperation is the name of the current operation that is used for the error reporting
	CurrentOperation string
}

// NewInstallationOperationFromOperation creates a new installation operation from an existing common operation.
// DEPRECATED: use the builder instead.
func NewInstallationOperationFromOperation(ctx context.Context, op *lsoperation.Operation, inst *InstallationImportsAndBlueprint, _ *types.UnstructuredTypedObject) (*Operation, error) {
	return NewOperationBuilder(inst).
		WithOperation(op).
		Build(ctx)
}

func (o *Operation) GetTargetImport(name string) *dataobjects.TargetExtension {
	return o.targets[name]
}
func (o *Operation) GetTargetListImport(name string) *dataobjects.TargetExtensionList {
	return o.targetLists[name]
}
func (o *Operation) GetTargetMapImport(name string) *dataobjects.TargetMapExtension {
	return o.targetMaps[name]
}
func (o *Operation) SetTargetImports(data map[string]*dataobjects.TargetExtension) {
	o.targets = data
}
func (o *Operation) SetTargetListImports(data map[string]*dataobjects.TargetExtensionList) {
	o.targetLists = data
}
func (o *Operation) SetTargetMapImports(data map[string]*dataobjects.TargetMapExtension) {
	o.targetMaps = data
}

// ResolveComponentDescriptors resolves the effective component descriptors for the installation.
// DEPRECATED: only used for tests. use the builder methods instead.
func (o *Operation) ResolveComponentDescriptors(ctx context.Context) error {
	componentVersion, err := ResolveComponentDescriptor(ctx, o.ComponentsRegistry(), o.Inst.GetInstallation(), o.Context().External.Overwriter)
	if err != nil {
		return err
	}

	dependentComponentVersions, err := model.GetTransitiveComponentReferences(ctx,
		componentVersion,
		o.Context().External.RepositoryContext,
		o.Context().External.Overwriter)
	if err != nil {
		return err
	}

	o.ComponentVersion = componentVersion
	o.ResolvedComponentDescriptorList = dependentComponentVersions
	return nil
}

// Context returns the context of the operated installation
func (o *Operation) Context() *Scope {
	return &o.context
}

// InstallationContextName returns the name of the current installation context.
func (o *Operation) InstallationContextName() string {
	return o.context.Name
}

// JSONSchemaValidator returns a jsonschema validator.
func (o *Operation) JSONSchemaValidator(schema []byte) (*jsonschema.Validator, error) {
	v := jsonschema.NewValidator(&jsonschema.ReferenceContext{
		LocalTypes:        o.Inst.GetBlueprint().Info.LocalTypes,
		BlueprintFs:       o.Inst.GetBlueprint().Fs,
		ComponentVersion:  o.ComponentVersion,
		RegistryAccess:    o.ComponentsRegistry(),
		RepositoryContext: o.context.External.RepositoryContext,
	})
	err := v.CompileSchema(schema)
	if err != nil {
		return nil, fmt.Errorf("error compiling jsonschema: %w", err)
	}
	return v, nil
}

// ListSubinstallations returns a list of all subinstallations of the given installation.
// Returns nil if no installations can be found
func (o *Operation) ListSubinstallations(ctx context.Context, subInstCache *lsv1alpha1.SubInstCache,
	readID read_write_layer.ReadID) ([]*lsv1alpha1.Installation, error) {

	return ListSubinstallations(ctx, o.LsUncachedClient(), o.Inst.GetInstallation(), subInstCache, readID)
}

type FilterInstallationFunc func(inst *lsv1alpha1.Installation) bool

// ListSubinstallations returns a list of all subinstallations of the given installation.
// The returned subinstallations can be filtered
// Returns nil if no installations can be found.
func ListSubinstallations(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation,
	subInstCache *lsv1alpha1.SubInstCache, readID read_write_layer.ReadID, filter ...FilterInstallationFunc) ([]*lsv1alpha1.Installation, error) {

	tmpInstallations := []*lsv1alpha1.Installation{}

	if subInstCache != nil {
		for i := range subInstCache.OrphanedSubs {
			nextInst := &lsv1alpha1.Installation{}
			key := client.ObjectKey{Namespace: inst.Namespace, Name: subInstCache.OrphanedSubs[i]}
			if err := read_write_layer.GetInstallation(ctx, kubeClient, key, nextInst, readID); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			tmpInstallations = append(tmpInstallations, nextInst)
		}

		for i := range subInstCache.ActiveSubs {
			nextInst := &lsv1alpha1.Installation{}
			key := client.ObjectKey{Namespace: inst.Namespace, Name: subInstCache.ActiveSubs[i].ObjectName}
			if err := read_write_layer.GetInstallation(ctx, kubeClient, key, nextInst, readID); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			tmpInstallations = append(tmpInstallations, nextInst)
		}
	} else {
		installationList := &lsv1alpha1.InstallationList{}

		err := read_write_layer.ListInstallations(ctx, kubeClient, installationList, readID,
			client.InNamespace(inst.Namespace),
			client.MatchingLabels{
				lsv1alpha1.EncompassedByLabel: inst.Name,
			})

		if err != nil {
			return nil, err
		}
		if len(installationList.Items) == 0 {
			return nil, nil
		}

		for i := range installationList.Items {
			tmpInstallations = append(tmpInstallations, &installationList.Items[i])
		}
	}

	// the controller-runtime cache does currently not support field selectors (except a simple equal matcher).
	// Therefore, we have to use our own filtering.
	filterInst := func(inst *lsv1alpha1.Installation) bool {
		for _, f := range filter {
			if f(inst) {
				return true
			}
		}
		return false
	}

	installations := make([]*lsv1alpha1.Installation, 0)
	for i := range tmpInstallations {
		if len(filter) != 0 && filterInst(tmpInstallations[i]) {
			continue
		}
		installations = append(installations, tmpInstallations[i])
	}
	return installations, nil
}

// UpdateInstallationStatus updates the status of a installation
func (o *Operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.Installation, writeID read_write_layer.WriteID,
	updatedConditions ...lsv1alpha1.Condition) error {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	if err := o.WriterToLsUncachedClient().UpdateInstallationStatus(ctx, writeID, inst); err != nil {
		logger.Error(err, "unable to set installation status")
		return err
	}
	return nil
}

// GetImportedDataObjects returns all imported data objects of the installation.
// It also updates the imports.
func (o *Operation) GetImportedDataObjects(ctx context.Context) (map[string]*dataobjects.DataObject, error) {
	dataObjects := map[string]*dataobjects.DataObject{}
	for _, def := range o.Inst.GetInstallation().Spec.Imports.Data {

		do, _, err := GetDataImport(ctx, o.LsUncachedClient(), o.Context().Name, &o.Inst.InstallationAndImports, def)
		if err != nil {
			return nil, err
		}
		dataObjects[def.Name] = do

		var (
			sourceRef *lsv1alpha1.ObjectReference
			owner     = kutil.GetOwner(do.Raw.ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.GetInstallation()) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.GetInstallation().Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := read_write_layer.GetInstallation(ctx, o.LsUncachedClient(), sourceRef.NamespacedName(), inst, read_write_layer.R000008); err != nil {
				return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
					sourceRef.NamespacedName().String(), def.Name, err)
			}
		}
	}

	return dataObjects, nil
}

// GetImportedTargets returns all imported targets of the installation.
func (o *Operation) GetImportedTargets(ctx context.Context) (map[string]*dataobjects.TargetExtension, error) {
	targets := map[string]*dataobjects.TargetExtension{}
	for _, def := range o.Inst.GetInstallation().Spec.Imports.Targets {
		if len(def.Target) == 0 {
			// It's a target list, skip it
			continue
		}
		target, err := GetTargetImport(ctx, o.LsUncachedClient(), o.Context().Name, o.Inst.GetInstallation(), def)
		if err != nil {
			return nil, err
		}
		targets[def.Name] = target

		var (
			sourceRef *lsv1alpha1.ObjectReference
			owner     = kutil.GetOwner(target.GetTarget().ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.GetInstallation()) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.GetInstallation().Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := read_write_layer.GetInstallation(ctx, o.LsUncachedClient(), sourceRef.NamespacedName(), inst,
				read_write_layer.R000004); err != nil {
				return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
					sourceRef.NamespacedName().String(), def.Name, err)
			}
		}
	}

	return targets, nil
}

// GetImportedTargetLists returns all imported target lists of the installation.
func (o *Operation) GetImportedTargetLists(ctx context.Context) (map[string]*dataobjects.TargetExtensionList, error) {
	targets := map[string]*dataobjects.TargetExtensionList{}
	for _, def := range o.Inst.GetInstallation().Spec.Imports.Targets {
		if len(def.Target) != 0 || def.TargetMap != nil || len(def.TargetMapReference) > 0 {
			// It's a single target or a target map, skip it
			continue
		}
		var (
			tl  *dataobjects.TargetExtensionList
			err error
		)
		if def.Targets != nil {
			// List of target names
			tl, err = GetTargetListImportByNames(ctx, o.LsUncachedClient(), o.Context().Name, o.Inst.GetInstallation(), def)
		} else if len(def.TargetListReference) != 0 {
			// TargetListReference is converted to a label selector internally
			tl, err = GetTargetListImportBySelector(ctx, o.LsUncachedClient(), o.Context().Name, o.Inst.GetInstallation(), map[string]string{lsv1alpha1.DataObjectKeyLabel: def.TargetListReference}, def)
		} else {
			// Invalid target
			err = fmt.Errorf("invalid target definition '%s': none of target, targets and targetListRef is defined", def.Name)
		}
		if err != nil {
			return nil, err
		}

		targets[def.Name] = tl
	}

	return targets, nil
}

// GetImportedTargetMaps returns all imported target maps of the installation.
func (o *Operation) GetImportedTargetMaps(ctx context.Context) (map[string]*dataobjects.TargetMapExtension, error) {
	targetMaps := map[string]*dataobjects.TargetMapExtension{}

	for _, def := range o.Inst.GetInstallation().Spec.Imports.Targets {
		if len(def.Target) != 0 || def.Targets != nil || len(def.TargetListReference) != 0 {
			// It's a target or target list, skip it
			continue
		}

		var tm *dataobjects.TargetMapExtension
		var err error
		if def.TargetMap != nil {
			tm, err = GetTargetMapImportByNames(ctx, o.LsUncachedClient(), o.Context().Name, o.Inst.GetInstallation(), def)
		} else if len(def.TargetMapReference) != 0 {
			tm, err = GetTargetMapImportBySelector(ctx, o.LsUncachedClient(), o.Context().Name, o.Inst.GetInstallation(), def)
		} else {
			err = fmt.Errorf("invalid target definition %s", def.Name)
		}
		if err != nil {
			return nil, err
		}

		targetMaps[def.Name] = tm
	}

	return targetMaps, nil
}

// NewError creates a new error with the current operation
func (o *Operation) NewError(err error, reason, message string, codes ...lsv1alpha1.ErrorCode) lserrors.LsError {
	return lserrors.NewWrappedError(err, o.CurrentOperation, reason, message, codes...)
}

// CreateEventFromCondition creates a new event based on the given condition
func (o *Operation) CreateEventFromCondition(ctx context.Context, inst *lsv1alpha1.Installation, cond lsv1alpha1.Condition) error {
	o.Operation.EventRecorder().Event(inst, corev1.EventTypeWarning, cond.Reason, cond.Message)
	return nil
}

// GetRootInstallations returns all root installations in the system.
// Keep in mind that root installation might not set a component repository context.
func GetRootInstallations(ctx context.Context, kubeClient client.Client, filter func(lsv1alpha1.Installation) bool, opts ...client.ListOption) ([]*lsv1alpha1.Installation, error) {
	r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)})

	installationList := &lsv1alpha1.InstallationList{}
	if err := read_write_layer.ListInstallations(ctx, kubeClient, installationList, read_write_layer.R000016, opts...); err != nil {
		return nil, err
	}

	installations := make([]*lsv1alpha1.Installation, 0)
	for _, obj := range installationList.Items {
		if filter != nil && filter(obj) {
			continue
		}
		inst := obj
		installations = append(installations, &inst)
	}
	return installations, nil
}

// CreateOrUpdateExports creates or updates the data objects that holds the exported values of the installation.
func (o *Operation) CreateOrUpdateExports(ctx context.Context, dataExports []*dataobjects.DataObject, targetExports []*dataobjects.TargetExtension) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateExportsCondition)

	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.GetInstallation())
	for _, do := range dataExports {
		do = do.
			SetNamespace(o.Inst.GetInstallation().Namespace).
			SetSource(src).
			SetContext(o.InstallationContextName()).
			SetJobID(o.Inst.GetInstallation().Status.JobID)
		raw, err := do.Build()
		if err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"CreateDataObjects",
					fmt.Sprintf("unable to create data object for export %s", do.Metadata.Key)))
			return fmt.Errorf("unable to build data object for export %s: %w", do.Metadata.Key, err)
		}

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000068, raw, func() error {
			if err, err2 := lsutil.SetExclusiveOwnerReference(o.Inst.GetInstallation(), raw); err != nil {
				return fmt.Errorf("dataobject '%s' for export '%s' conflicts with existing dataobject owned by another installation: %w", client.ObjectKeyFromObject(raw).String(), do.Metadata.Key, err)
			} else if err2 != nil {
				return fmt.Errorf("error setting owner reference: %w", err2)
			}
			return do.Apply(raw)
		}); err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateDataObjects",
					fmt.Sprintf("unable to create data object for export %s", do.Metadata.Key)))
			return fmt.Errorf("unable to create or update data object %s for export %s: %w", raw.Name, do.Metadata.Key, err)
		}
	}

	for _, target := range targetExports {
		target = target.
			SetNamespace(o.Inst.GetInstallation().Namespace).
			SetSource(src).
			SetContext(o.InstallationContextName()).
			SetJobID(o.Inst.GetInstallation().Status.JobID)

		targetForUpdate := &lsv1alpha1.Target{}
		target.ApplyNameAndNamespace(targetForUpdate)

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000069, targetForUpdate, func() error {
			if err, err2 := lsutil.SetExclusiveOwnerReference(o.Inst.GetInstallation(), targetForUpdate); err != nil {
				return fmt.Errorf("target object '%s' for export '%s' conflicts with existing target owned by another installation: %w",
					client.ObjectKeyFromObject(targetForUpdate).String(), target.GetMetadata().Key, err)
			} else if err2 != nil {
				return fmt.Errorf("error setting owner reference: %w", err2)
			}
			return target.Apply(targetForUpdate)
		}); err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateTargets",
					fmt.Sprintf("unable to create target for export %s", target.GetMetadata().Key)))
			return fmt.Errorf("unable to create or update target %s for export %s: %w", targetForUpdate.Name, target.GetMetadata().Key, err)
		}
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue, "DataObjectsCreated", "DataObjects successfully created")
	return o.UpdateInstallationStatus(ctx, o.Inst.GetInstallation(), read_write_layer.W000057, cond)
}

// CreateOrUpdateImports creates or updates the data objects that holds the imported values for every import
func (o *Operation) CreateOrUpdateImports(ctx context.Context) error {
	return o.createOrUpdateImports(ctx, o.Inst.GetBlueprint().Info.Imports)
}

func (o *Operation) createOrUpdateImports(ctx context.Context, importDefs lsv1alpha1.ImportDefinitionList) error {
	importedValues := o.Inst.GetImports()
	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.GetInstallation())
	for _, importDef := range importDefs {
		importData, ok := importedValues[importDef.Name]
		if !ok {
			// todo: create test for optional imports
			if importDef.Required != nil && !*importDef.Required {
				continue
			}
			return fmt.Errorf("import %s not defined", importDef.Name)
		}

		if len(importDef.ConditionalImports) > 0 {
			if err := o.createOrUpdateImports(ctx, importDef.ConditionalImports); err != nil {
				return err
			}
		}

		switch importDef.Type {
		case lsv1alpha1.ImportTypeData:
			if err := o.createOrUpdateDataImport(ctx, src, importDef, importData); err != nil {
				return fmt.Errorf("unable to create or update data import '%s': %w", importDef.Name, err)
			}
		case lsv1alpha1.ImportTypeTarget:
			if err := o.createOrUpdateTargetImport(ctx, src, importDef, importData); err != nil {
				return fmt.Errorf("unable to create or update target import '%s': %w", importDef.Name, err)
			}
		case lsv1alpha1.ImportTypeTargetList:
			importDataList, ok2 := importData.([]interface{})
			if !ok2 {
				return fmt.Errorf("targetlist import '%s' is not a list", importDef.Name)
			}
			if err := o.createOrUpdateTargetListImport(ctx, src, importDef, importDataList); err != nil {
				return fmt.Errorf("unable to create or update targetlist import '%s': %w", importDef.Name, err)
			}
		case lsv1alpha1.ImportTypeTargetMap:
			importTargetMap, ok := importData.(map[string]interface{})
			if !ok {
				return fmt.Errorf("targetmap import '%s' is not a map", importDef.Name)
			}
			if err := o.createOrUpdateTargetMapImport(ctx, src, importDef, importTargetMap); err != nil {
				return fmt.Errorf("unable to create or update targetmap import '%s': %w", importDef.Name, err)
			}
		default:
			return fmt.Errorf("unknown import type '%s' for import '%s'", string(importDef.Type), importDef.Name)
		}

	}
	return nil
}

func (o *Operation) createOrUpdateDataImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, importData interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateImportsCondition)
	do := dataobjects.New().
		SetNamespace(o.Inst.GetInstallation().Namespace).SetSource(src).
		SetContext(src).
		SetKey(importDef.Name).SetSourceType(lsv1alpha1.ImportDataObjectSourceType).
		SetData(importData).
		SetJobID(o.Inst.GetInstallation().Status.JobID)
	raw, err := do.Build()
	if err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to build data object for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000070, raw, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.GetInstallation(), raw, api.LandscaperScheme); err != nil {
			return err
		}
		return do.Apply(raw)
	}); err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to create or update data object '%s' for import '%s': %w", raw.Name, importDef.Name, err)
	}
	return nil
}

func (o *Operation) createOrUpdateTargetImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, values interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateImportsCondition)
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	target := &lsv1alpha1.Target{}
	if _, _, err := api.Decoder.Decode(data, nil, target); err != nil {
		return err
	}
	targetExtension := dataobjects.NewTargetExtension(target, nil)

	targetExtension.SetNamespace(o.Inst.GetInstallation().Namespace).
		SetContext(src).
		SetKey(importDef.Name).
		SetIndex(nil).
		SetTargetMapKey(nil).
		SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType).
		SetJobID(o.Inst.GetInstallation().Status.JobID)

	targetForUpdate := &lsv1alpha1.Target{}
	targetExtension.ApplyNameAndNamespace(targetForUpdate)

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000071, targetForUpdate, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.GetInstallation(), targetForUpdate, api.LandscaperScheme); err != nil {
			return err
		}
		return targetExtension.Apply(targetForUpdate)
	}); err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create target for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to create or update target '%s' for import '%s': %w", targetForUpdate.Name, importDef.Name, err)
	}

	return nil
}

func (o *Operation) createOrUpdateTargetListImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, values []interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateImportsCondition)
	tars := make([]lsv1alpha1.Target, len(values))
	for i := range values {
		tar := &lsv1alpha1.Target{}
		data, err := json.Marshal(values[i])
		if err != nil {
			return err
		}
		if _, _, err := api.Decoder.Decode(data, nil, tar); err != nil {
			return err
		}
		tars[i] = *tar
	}
	targetExtensionList := dataobjects.NewTargetExtensionList(tars, nil)
	for i := range targetExtensionList.GetTargetExtensions() {
		tar := targetExtensionList.GetTargetExtensions()[i]
		tar.SetNamespace(o.Inst.GetInstallation().Namespace).
			SetContext(src).
			SetKey(importDef.Name).
			SetIndex(ptr.To[int](i)).
			SetTargetMapKey(nil).
			SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType).
			SetJobID(o.Inst.GetInstallation().Status.JobID)
	}

	targets, err := targetExtensionList.Build(importDef.Name)
	if err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create targets for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to build targets for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	for i, target := range targets {
		tmpTarget := &lsv1alpha1.Target{ObjectMeta: metav1.ObjectMeta{Namespace: target.Namespace, Name: target.Name}}
		if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000072, tmpTarget, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.GetInstallation(), tmpTarget, api.LandscaperScheme); err != nil {
				return err
			}
			return targetExtensionList.Apply(tmpTarget, i)
		}); err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"CreateTargets",
					fmt.Sprintf("unable to create target for import '%s'", importDef.Name)))
			return fmt.Errorf("unable to create or update target '%s' for import '%s': %w", target.Name, importDef.Name, err)
		}
	}

	return nil
}

func (o *Operation) createOrUpdateTargetMapImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, values map[string]interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateImportsCondition)

	tars := make(map[string]lsv1alpha1.Target)
	for key := range values {
		tar := &lsv1alpha1.Target{}
		data, err := json.Marshal(values[key])
		if err != nil {
			return err
		}
		if _, _, err := api.Decoder.Decode(data, nil, tar); err != nil {
			return err
		}
		tars[key] = *tar
	}

	targetMapExtension := dataobjects.NewTargetMapExtension(tars, nil)
	for key := range targetMapExtension.GetTargetExtensions() {
		tar := targetMapExtension.GetTargetExtensions()[key]
		tar.SetNamespace(o.Inst.GetInstallation().Namespace).
			SetContext(src).
			SetKey(importDef.Name).
			SetIndex(nil).
			SetTargetMapKey(ptr.To(key)).
			SetSource(src).
			SetSourceType(lsv1alpha1.ImportDataObjectSourceType).
			SetJobID(o.Inst.GetInstallation().Status.JobID)
	}

	targets, err := targetMapExtension.Build(importDef.Name)
	if err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateTargets",
				fmt.Sprintf("unable to create targets for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to build targets for import '%s': %w", importDef.Name, err)
	}

	for targetMapKey, target := range targets {
		tmpTarget := &lsv1alpha1.Target{ObjectMeta: metav1.ObjectMeta{Namespace: target.Namespace, Name: target.Name}}
		if _, err := o.WriterToLsUncachedClient().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000089, tmpTarget, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.GetInstallation(), tmpTarget, api.LandscaperScheme); err != nil {
				return err
			}
			return targetMapExtension.Apply(tmpTarget, targetMapKey)
		}); err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateTargets",
					fmt.Sprintf("unable to create target for targetmap import '%s'", importDef.Name)))
			return fmt.Errorf("unable to create or update target '%s' for targetmap import '%s': %w", target.Name, importDef.Name, err)
		}
	}

	return nil
}

// GetExportForKey creates a dataobject from a dataobject
func (o *Operation) GetExportForKey(ctx context.Context, key string) (*dataobjects.DataObject, error) {
	doName := lsv1alpha1helper.GenerateDataObjectName(o.context.Name, key)
	rawDO := &lsv1alpha1.DataObject{}
	if err := o.LsUncachedClient().Get(ctx, kutil.ObjectKey(doName, o.Inst.GetInstallation().Namespace), rawDO); err != nil {
		return nil, err
	}
	return dataobjects.NewFromDataObject(rawDO)
}
