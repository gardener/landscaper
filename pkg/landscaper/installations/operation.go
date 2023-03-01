// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lserrors "github.com/gardener/landscaper/apis/errors"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	lsutil "github.com/gardener/landscaper/pkg/utils"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// Operation contains all installation operations and implements the Operation interface.
type Operation struct {
	*lsoperation.Operation

	Inst                            *InstallationImportsAndBlueprint
	ComponentDescriptor             *cdv2.ComponentDescriptor
	BlobResolver                    ctf.BlobResolver
	ResolvedComponentDescriptorList *cdv2.ComponentDescriptorList
	context                         Scope

	targetLists map[string]*dataobjects.TargetExtensionList
	targets     map[string]*dataobjects.TargetExtension

	// CurrentOperation is the name of the current operation that is used for the error reporting
	CurrentOperation string
}

// NewInstallationOperationFromOperation creates a new installation operation from an existing common operation.
// DEPRECATED: use the builder instead.
func NewInstallationOperationFromOperation(ctx context.Context, op *lsoperation.Operation, inst *InstallationImportsAndBlueprint, _ *cdv2.UnstructuredTypedObject) (*Operation, error) {
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
func (o *Operation) SetTargetImports(data map[string]*dataobjects.TargetExtension) {
	o.targets = data
}
func (o *Operation) SetTargetListImports(data map[string]*dataobjects.TargetExtensionList) {
	o.targetLists = data
}

// ResolveComponentDescriptors resolves the effective component descriptors for the installation.
// DEPRECATED: only used for tests. use the builder methods instead.
func (o *Operation) ResolveComponentDescriptors(ctx context.Context) error {
	cd, blobResolver, err := ResolveComponentDescriptor(ctx, o.ComponentsRegistry(), o.Inst.GetInstallation(), o.Context().External.Overwriter)
	if err != nil {
		return err
	}
	if cd == nil {
		return nil
	}

	resolvedCD, err := cdutils.ResolveToComponentDescriptorList(ctx, o.ComponentsRegistry(), *cd, o.Context().External.RepositoryContext, o.Context().External.Overwriter)
	if err != nil {
		return err
	}
	o.ComponentDescriptor = cd
	o.BlobResolver = blobResolver
	o.ResolvedComponentDescriptorList = &resolvedCD
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
		LocalTypes:          o.Inst.GetBlueprint().Info.LocalTypes,
		BlueprintFs:         o.Inst.GetBlueprint().Fs,
		ComponentDescriptor: o.ComponentDescriptor,
		ComponentResolver:   o.ComponentsRegistry(),
		RepositoryContext:   o.context.External.RepositoryContext,
	})
	err := v.CompileSchema(schema)
	if err != nil {
		return nil, fmt.Errorf("error compiling jsonschema: %w", err)
	}
	return v, nil
}

// ListSubinstallations returns a list of all subinstallations of the given installation.
// Returns nil if no installations can be found
func (o *Operation) ListSubinstallations(ctx context.Context, filter ...FilterInstallationFunc) ([]*lsv1alpha1.Installation, error) {
	return ListSubinstallations(ctx, o.Client(), o.Inst.GetInstallation(), filter...)
}

type FilterInstallationFunc func(inst *lsv1alpha1.Installation) bool

// ListSubinstallations returns a list of all subinstallations of the given installation.
// The returned subinstallations can be filtered
// Returns nil if no installations can be found.
func ListSubinstallations(ctx context.Context, kubeClient client.Client, inst *lsv1alpha1.Installation, filter ...FilterInstallationFunc) ([]*lsv1alpha1.Installation, error) {
	installationList := &lsv1alpha1.InstallationList{}

	// the controller-runtime cache does currently not support field selectors (except a simple equal matcher).
	// Therefore, we have to use our own filtering.
	err := read_write_layer.ListInstallations(ctx, kubeClient, installationList, client.InNamespace(inst.Namespace),
		client.MatchingLabels{
			lsv1alpha1.EncompassedByLabel: inst.Name,
		})
	if err != nil {
		return nil, err
	}
	if len(installationList.Items) == 0 {
		return nil, nil
	}

	filterInst := func(inst *lsv1alpha1.Installation) bool {
		for _, f := range filter {
			if f(inst) {
				return true
			}
		}
		return false
	}
	installations := make([]*lsv1alpha1.Installation, 0)
	for _, inst := range installationList.Items {
		if len(filter) != 0 && filterInst(&inst) {
			continue
		}
		installations = append(installations, inst.DeepCopy())
	}
	return installations, nil
}

// UpdateInstallationStatus updates the status of a installation
func (o *Operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.Installation,
	updatedConditions ...lsv1alpha1.Condition) error {

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(inst).String()})

	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	if err := o.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000018, inst); err != nil {
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

		do, _, err := GetDataImport(ctx, o.Client(), o.Context().Name, &o.Inst.InstallationAndImports, def)
		if err != nil {
			return nil, err
		}
		dataObjects[def.Name] = do

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = dataobjects.ImportedBase(do).ComputeConfigGeneration()
			owner     = kutil.GetOwner(do.Raw.ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.GetInstallation()) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.GetInstallation().Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
				return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
					sourceRef.NamespacedName().String(), def.Name, err)
			}
			configGen = inst.Status.ConfigGeneration
		}

		importStatus := lsv1alpha1.ImportStatus{
			Name:             def.Name,
			Type:             lsv1alpha1.DataImportStatusType,
			DataRef:          def.DataRef,
			SourceRef:        sourceRef,
			ConfigGeneration: configGen,
		}
		if len(def.DataRef) != 0 {
			importStatus.DataRef = def.DataRef
		} else if def.SecretRef != nil {
			secretRef := lsutil.SecretRefFromLocalRef(def.SecretRef, o.Inst.GetInstallation().Namespace)
			importStatus.SecretRef = fmt.Sprintf("%s#%s", secretRef.NamespacedName().String(), secretRef.Key)
		} else if def.ConfigMapRef != nil {
			configMapRef := lsutil.ConfigMapRefFromLocalRef(def.ConfigMapRef, o.Inst.GetInstallation().Namespace)
			importStatus.ConfigMapRef = fmt.Sprintf("%s#%s", configMapRef.NamespacedName().String(), configMapRef.Key)
		}

		o.Inst.ImportStatus().Update(importStatus)
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
		target, err := GetTargetImport(ctx, o.Client(), o.Context().Name, o.Inst.GetInstallation(), def)
		if err != nil {
			return nil, err
		}
		targets[def.Name] = target

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = dataobjects.ImportedBase(target).ComputeConfigGeneration()
			owner     = kutil.GetOwner(target.GetTarget().ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.GetInstallation()) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.GetInstallation().Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
				return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
					sourceRef.NamespacedName().String(), def.Name, err)
			}
			configGen = inst.Status.ConfigGeneration
		}
		o.Inst.ImportStatus().Update(lsv1alpha1.ImportStatus{
			Name:             def.Name,
			Type:             lsv1alpha1.TargetImportStatusType,
			Target:           def.Target,
			SourceRef:        sourceRef,
			ConfigGeneration: configGen,
		})
	}

	return targets, nil
}

// GetImportedTargetLists returns all imported target lists of the installation.
func (o *Operation) GetImportedTargetLists(ctx context.Context) (map[string]*dataobjects.TargetExtensionList, error) {
	targets := map[string]*dataobjects.TargetExtensionList{}
	for _, def := range o.Inst.GetInstallation().Spec.Imports.Targets {
		if len(def.Target) != 0 {
			// It's a single target, skip it
			continue
		}
		var (
			tl  *dataobjects.TargetExtensionList
			err error
		)
		if def.Targets != nil {
			// List of target names
			tl, err = GetTargetListImportByNames(ctx, o.Client(), o.Context().Name, o.Inst.GetInstallation(), def)
		} else if len(def.TargetListReference) != 0 {
			// TargetListReference is converted to a label selector internally
			tl, err = GetTargetListImportBySelector(ctx, o.Client(), o.Context().Name, o.Inst.GetInstallation(), map[string]string{lsv1alpha1.DataObjectKeyLabel: def.TargetListReference}, def, true)
		} else {
			// Invalid target
			err = fmt.Errorf("invalid target definition '%s': none of target, targets and targetListRef is defined", def.Name)
		}
		if err != nil {
			return nil, err
		}

		targets[def.Name] = tl

		tis := make([]lsv1alpha1.TargetImportStatus, len(tl.GetTargetExtensions()))
		for i, t := range tl.GetTargetExtensions() {
			var (
				sourceRef *lsv1alpha1.ObjectReference
				configGen = dataobjects.ImportedBase(t).ComputeConfigGeneration()
				owner     = kutil.GetOwner(t.GetTarget().ObjectMeta)
			)
			if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.GetInstallation()) {
				sourceRef = &lsv1alpha1.ObjectReference{
					Name:      owner.Name,
					Namespace: o.Inst.GetInstallation().Namespace,
				}
				inst := &lsv1alpha1.Installation{}
				if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
					return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
						sourceRef.NamespacedName().String(), def.Name, err)
				}
				configGen = inst.Status.ConfigGeneration
			}
			tis[i] = lsv1alpha1.TargetImportStatus{
				Target:           t.GetTarget().Name,
				SourceRef:        sourceRef,
				ConfigGeneration: configGen,
			}
		}
		o.Inst.ImportStatus().Update(lsv1alpha1.ImportStatus{
			Name:    def.Name,
			Type:    lsv1alpha1.TargetListImportStatusType,
			Targets: tis,
		})
	}

	return targets, nil
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
	if err := read_write_layer.ListInstallations(ctx, kubeClient, installationList, opts...); err != nil {
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

// NewTriggerDependents triggers all installations that depend on the current installation.
func (o *Operation) NewTriggerDependents(ctx context.Context) error {
	for _, sibling := range o.Context().Siblings {
		if !importsAnyExport(o.Inst, sibling) {
			continue
		}

		if IsRootInstallation(o.Inst.GetInstallation()) {
			metav1.SetMetaDataAnnotation(&sibling.GetInstallation().ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		} else {
			lsv1alpha1helper.Touch(&sibling.GetInstallation().ObjectMeta)
		}

		if err := o.Writer().UpdateInstallation(ctx, read_write_layer.W000085, sibling.GetInstallation()); err != nil {
			return err
		}
	}
	return nil
}

// SetExportConfigGeneration returns the new export generation of the installation
// based on its own generation and its context
func (o *Operation) SetExportConfigGeneration(ctx context.Context) error {
	// we have to set our config generation to the desired state

	o.Inst.GetInstallation().Status.ConfigGeneration = ""
	return o.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000016, o.Inst.GetInstallation())
}

// CreateOrUpdateExports creates or updates the data objects that holds the exported values of the installation.
func (o *Operation) CreateOrUpdateExports(ctx context.Context, dataExports []*dataobjects.DataObject, targetExports []*dataobjects.TargetExtension) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.GetInstallation().Status.Conditions, lsv1alpha1.CreateExportsCondition)

	configGen, err := CreateGenerationHash(o.Inst.GetInstallation())
	if err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateConfigHash",
				fmt.Sprintf("unable to create config hash: %s", err.Error())))
		return err
	}

	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.GetInstallation())
	for _, do := range dataExports {
		do = do.SetNamespace(o.Inst.GetInstallation().Namespace).SetSource(src).SetContext(o.InstallationContextName())
		raw, err := do.Build()
		if err != nil {
			o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"CreateDataObjects",
					fmt.Sprintf("unable to create data object for export %s", do.Metadata.Key)))
			return fmt.Errorf("unable to build data object for export %s: %w", do.Metadata.Key, err)
		}

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := o.Writer().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000068, raw, func() error {
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
		target = target.SetNamespace(o.Inst.GetInstallation().Namespace).SetSource(src).SetContext(o.InstallationContextName())

		targetForUpdate := &lsv1alpha1.Target{}
		target.ApplyNameAndNamespace(targetForUpdate)

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000069, targetForUpdate, func() error {
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

	o.Inst.GetInstallation().Status.ConfigGeneration = configGen
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue, "DataObjectsCreated", "DataObjects successfully created")
	return o.UpdateInstallationStatus(ctx, o.Inst.GetInstallation(), cond)
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
		SetData(importData)
	raw, err := do.Build()
	if err != nil {
		o.Inst.GetInstallation().Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.GetInstallation().Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to build data object for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.Writer().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000070, raw, func() error {
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
		SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType)

	targetForUpdate := &lsv1alpha1.Target{}
	targetExtension.ApplyNameAndNamespace(targetForUpdate)

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000071, targetForUpdate, func() error {
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
			SetIndex(pointer.Int(i)).
			SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType)
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
		if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000072, target, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.GetInstallation(), target, api.LandscaperScheme); err != nil {
				return err
			}
			return targetExtensionList.Apply(target, i)
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

// GetExportForKey creates a dataobject from a dataobject
func (o *Operation) GetExportForKey(ctx context.Context, key string) (*dataobjects.DataObject, error) {
	doName := lsv1alpha1helper.GenerateDataObjectName(o.context.Name, key)
	rawDO := &lsv1alpha1.DataObject{}
	if err := o.Client().Get(ctx, kutil.ObjectKey(doName, o.Inst.GetInstallation().Namespace), rawDO); err != nil {
		return nil, err
	}
	return dataobjects.NewFromDataObject(rawDO)
}

func importsAnyExport(exporter *InstallationImportsAndBlueprint, importer *InstallationAndImports) bool {
	for _, export := range exporter.GetInstallation().Spec.Exports.Data {
		for _, def := range importer.GetInstallation().Spec.Imports.Data {
			if def.DataRef == export.DataRef {
				return true
			}
		}
	}
	for _, export := range exporter.GetInstallation().Spec.Exports.Targets {
		for _, def := range importer.GetInstallation().Spec.Imports.Targets {
			if def.Target == export.Target {
				return true
			}
		}
	}
	return false
}
