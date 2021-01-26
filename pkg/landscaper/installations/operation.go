// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/jsonschema"
	"github.com/gardener/landscaper/pkg/landscaper/registry/components/cdutils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
)

// Operation contains all installation operations and implements the Operation interface.
type Operation struct {
	lsoperation.Interface

	Inst                            *Installation
	ComponentDescriptor             *cdv2.ComponentDescriptor
	BlobResolver                    ctf.BlobResolver
	ResolvedComponentDescriptorList *cdv2.ComponentDescriptorList
	context                         Context
}

// NewInstallationOperation creates a new installation operation
func NewInstallationOperation(ctx context.Context, log logr.Logger, c client.Client, scheme *runtime.Scheme, cRegistry ctf.ComponentResolver, inst *Installation) (*Operation, error) {
	return NewInstallationOperationFromOperation(ctx, lsoperation.NewOperation(log, c, scheme, cRegistry), inst)
}

// NewInstallationOperationFromOperation creates a new installation operation from an existing common operation
func NewInstallationOperationFromOperation(ctx context.Context, op lsoperation.Interface, inst *Installation) (*Operation, error) {
	instOp := &Operation{
		Interface: op,
		Inst:      inst,
	}

	if err := instOp.ResolveComponentDescriptors(ctx); err != nil {
		return nil, err
	}
	if err := instOp.SetInstallationContext(ctx); err != nil {
		return nil, err
	}
	return instOp, nil
}

// ResolveComponentDescriptors resolves the effective component descriptors for the installation.
func (o *Operation) ResolveComponentDescriptors(ctx context.Context) error {
	cd, blobResolver, err := ResolveComponentDescriptor(ctx, o.ComponentsRegistry(), o.Inst.Info)
	if err != nil {
		return err
	}
	if cd == nil {
		return nil
	}

	resolvedCD, err := cdutils.ResolveToComponentDescriptorList(ctx, o.ComponentsRegistry(), *cd)
	if err != nil {
		return err
	}
	o.ComponentDescriptor = cd
	o.BlobResolver = blobResolver
	o.ResolvedComponentDescriptorList = &resolvedCD
	return nil
}

// Context returns the context of the operated installation
func (o *Operation) Context() *Context {
	return &o.context
}

// InstallationContextName returns the name of the current installation context.
func (o *Operation) InstallationContextName() string {
	return o.context.Name
}

// JSONSchemaValidator returns a jsonschema validator for the current installation and blueprint.
func (o *Operation) JSONSchemaValidator() *jsonschema.Validator {
	return &jsonschema.Validator{
		Config: &jsonschema.LoaderConfig{
			LocalTypes:                 o.Inst.Blueprint.Info.LocalTypes,
			BlueprintFs:                o.Inst.Blueprint.Fs,
			BlobResolver:               o.BlobResolver,
			ComponentDescriptor:        o.ComponentDescriptor,
			ComponentReferenceResolver: cdutils.ComponentReferenceResolverFromList(o.ResolvedComponentDescriptorList),
		},
	}
}

// UpdateInstallationStatus updates the status of a installation
func (o *Operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.Installation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error {
	inst.Status.Phase = phase
	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	if err := o.Client().Status().Update(ctx, inst); err != nil {
		o.Log().Error(err, "unable to set installation status")
		return err
	}
	return nil
}

// GetImportedDataObjects returns all imported data objects of the installation.
// It also updates the imports.
func (o *Operation) GetImportedDataObjects(ctx context.Context) (map[string]*dataobjects.DataObject, error) {
	dataObjects := map[string]*dataobjects.DataObject{}
	for _, def := range o.Inst.Info.Spec.Imports.Data {

		do, _, err := GetDataImport(ctx, o, o.Context().Name, o.Inst, def)
		if err != nil {
			return nil, err
		}
		dataObjects[def.Name] = do

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = strconv.Itoa(int(do.Raw.Generation))
			owner     = kutil.GetOwner(do.Raw.ObjectMeta)
		)
		if owner != nil && owner.Kind == "Installation" {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.Info.Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := o.Client().Get(ctx, sourceRef.NamespacedName(), inst); err != nil {
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
			importStatus.SecretRef = fmt.Sprintf("%s#%s", def.SecretRef.NamespacedName().String(), def.SecretRef.Key)
		} else if def.ConfigMapRef != nil {
			importStatus.ConfigMapRef = fmt.Sprintf("%s#%s", def.ConfigMapRef.NamespacedName().String(), def.ConfigMapRef.Key)
		}

		o.Inst.ImportStatus().Update(importStatus)
	}

	return dataObjects, nil
}

// GetImportedTargets returns all imported targets of the installation.
func (o *Operation) GetImportedTargets(ctx context.Context) (map[string]*dataobjects.Target, error) {
	targets := map[string]*dataobjects.Target{}
	for _, def := range o.Inst.Info.Spec.Imports.Targets {
		target, _, err := GetTargetImport(ctx, o, o.Context().Name, o.Inst, def.Target)
		if err != nil {
			return nil, err
		}
		targets[def.Name] = target

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = strconv.Itoa(int(target.Raw.Generation))
			owner     = kutil.GetOwner(target.Raw.ObjectMeta)
		)
		if owner != nil && owner.Kind == "Installation" {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.Info.Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := o.Client().Get(ctx, sourceRef.NamespacedName(), inst); err != nil {
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

// CreateEventFromCondition creates a new event based on the given condition
func (o *Operation) CreateEventFromCondition(ctx context.Context, inst *lsv1alpha1.Installation, cond lsv1alpha1.Condition) error {
	event := &corev1.Event{}
	event.GenerateName = "inst-"
	event.Namespace = inst.Namespace
	event.Type = "Warning"
	event.Source = corev1.EventSource{
		Component: "landscaper", // todo: make configurable by the caller
	}
	event.InvolvedObject = corev1.ObjectReference{
		Kind:            inst.Kind,
		Namespace:       inst.Namespace,
		Name:            inst.Name,
		UID:             inst.UID,
		APIVersion:      inst.APIVersion,
		ResourceVersion: inst.ResourceVersion,
	}
	event.Reason = cond.Reason
	event.Message = cond.Message
	event.Action = string(cond.Type)

	if err := o.Client().Create(ctx, event); err != nil {
		o.Log().Error(err, "unable to set installation status")
		return err
	}
	return nil
}

// GetRootInstallations returns all root installations in the system
func (o *Operation) GetRootInstallations(ctx context.Context, filter func(lsv1alpha1.Installation) bool, opts ...client.ListOption) ([]*lsv1alpha1.Installation, error) {
	r, err := labels.NewRequirement(lsv1alpha1.EncompassedByLabel, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	opts = append(opts, client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*r)})

	installationList := &lsv1alpha1.InstallationList{}
	if err := o.Client().List(ctx, installationList, opts...); err != nil {
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

//// GetStaticData constructs the static data from the installation.
//func (o *Operation) GetStaticData(ctx context.Context) (map[string]interface{}, error) {
//	if o.staticData != nil {
//		return o.staticData, nil
//	}
//
//	if len(o.Inst.Info.Spec.StaticData) == 0 {
//		return nil, nil
//	}
//
//	data := make(map[string]interface{})
//	for _, source := range o.Inst.Info.Spec.StaticData {
//		if source.Value != nil {
//			var values map[string]interface{}
//			if err := yaml.Unmarshal(source.Value, &values); err != nil {
//				return nil, errors.Wrap(err, "unable to parse value into map")
//			}
//			data = utils.MergeMaps(data, values)
//			continue
//		}
//
//		if source.ValueFrom == nil {
//			continue
//		}
//
//		if source.ValueFrom.SecretKeyRef != nil {
//			values, err := GetDataFromSecretKeyRef(ctx, o.Client(), source.ValueFrom.SecretKeyRef, o.Inst.Info.Namespace)
//			if err != nil {
//				return nil, err
//			}
//			data = utils.MergeMaps(data, values)
//			continue
//		}
//		if source.ValueFrom.SecretLabelSelector != nil {
//			values, err := GetDataFromSecretLabelSelectorRef(ctx, o.Client(), source.ValueFrom.SecretLabelSelector, o.Inst.Info.Namespace)
//			if err != nil {
//				return nil, err
//			}
//			data = utils.MergeMaps(data, values)
//		}
//	}
//	o.staticData = data
//	return data, nil
//}

// TriggerDependants triggers all installations that depend on the current installation.
// These are most likely all installation that import a key which is exported by the current installation.
func (o *Operation) TriggerDependants(ctx context.Context) error {
	for _, sibling := range o.Context().Siblings {
		if !importsAnyExport(o.Inst, sibling) {
			continue
		}

		// todo: maybe use patch
		metav1.SetMetaDataAnnotation(&sibling.Info.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		if err := o.Client().Update(ctx, sibling.Info); err != nil {
			return errors.Wrapf(err, "unable to trigger installation %s", sibling.Info.Name)
		}
	}
	return nil
}

// SetExportConfigGeneration returns the new export generation of the installation
// based on its own generation and its context
func (o *Operation) SetExportConfigGeneration(ctx context.Context) error {
	// we have to set our config generation to the desired state

	o.Inst.Info.Status.ConfigGeneration = ""
	return o.Client().Status().Update(ctx, o.Inst.Info)
}

// CreateOrUpdateExports creates or updates the data objects that holds the exported values of the installation.
func (o *Operation) CreateOrUpdateExports(ctx context.Context, dataExports []*dataobjects.DataObject, targetExports []*dataobjects.Target) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.Info.Status.Conditions, lsv1alpha1.CreateExportsCondition)

	configGen, err := CreateGenerationHash(o.Inst.Info)
	if err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateConfigHash",
				fmt.Sprintf("unable to create config hash: %s", err.Error())))
		return err
	}

	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.Info)
	for _, do := range dataExports {
		do = do.SetNamespace(o.Inst.Info.Namespace).SetSource(src).SetContext(o.InstallationContextName())
		raw, err := do.Build()
		if err != nil {
			o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"CreateDataObjects",
					fmt.Sprintf("unable to create data object for export %s", do.Metadata.Key)))
			return fmt.Errorf("unable to build data object for export %s: %w", do.Metadata.Key, err)
		}

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), raw, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.Info, raw, kubernetes.LandscaperScheme); err != nil {
				return err
			}
			return do.Apply(raw)
		}); err != nil {
			o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateDataObjects",
					fmt.Sprintf("unable to create data object for export %s", do.Metadata.Key)))
			return fmt.Errorf("unable to create or update data object %s for export %s: %w", raw.Name, do.Metadata.Key, err)
		}
	}

	for _, target := range targetExports {
		target = target.SetNamespace(o.Inst.Info.Namespace).SetSource(src).SetContext(o.InstallationContextName())
		raw, err := target.Build()
		if err != nil {
			o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
					"CreateTargets",
					fmt.Sprintf("unable to create target for export %s", target.Metadata.Key)))
			return fmt.Errorf("unable to build target for export %s: %w", target.Metadata.Key, err)
		}

		// we do not need to set controller ownership as we anyway need a separate garbage collection.
		if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), raw, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.Info, raw, kubernetes.LandscaperScheme); err != nil {
				return err
			}
			return target.Apply(raw)
		}); err != nil {
			o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
				lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse, "CreateTargets",
					fmt.Sprintf("unable to create target for export %s", target.Metadata.Key)))
			return fmt.Errorf("unable to create or update target %s for export %s: %w", raw.Name, target.Metadata.Key, err)
		}
	}

	o.Inst.Info.Status.ConfigGeneration = configGen
	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue, "DataObjectsCreated", "DataObjects successfully created")
	return o.UpdateInstallationStatus(ctx, o.Inst.Info, o.Inst.Info.Status.Phase, cond)
}

// CreateOrUpdateImports creates or updates the data objects that holds the imported values for every import
func (o *Operation) CreateOrUpdateImports(ctx context.Context, importedValues map[string]interface{}) error {
	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.Info)
	for _, importDef := range o.Inst.Blueprint.Info.Imports {
		importData, ok := importedValues[importDef.Name]
		if !ok {
			// todo: create test for optional imports
			if importDef.Required != nil && !*importDef.Required {
				continue
			}
			return fmt.Errorf("import %s not defined", importDef.Name)
		}

		if len(importDef.TargetType) != 0 {
			if err := o.createOrUpdateTargetImport(ctx, src, importDef, importData); err != nil {
				return fmt.Errorf("unable to create or update target import '%s': %w", importDef.Name, err)
			}
			continue
		}

		if err := o.createOrUpdateDataImport(ctx, src, importDef, importData); err != nil {
			return fmt.Errorf("unable to create or update data import '%s': %w", importDef.Name, err)
		}
	}
	return nil
}

func (o *Operation) createOrUpdateDataImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, importData interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.Info.Status.Conditions, lsv1alpha1.CreateImportsCondition)
	do := dataobjects.New().
		SetNamespace(o.Inst.Info.Namespace).SetSource(src).
		SetContext(src).
		SetKey(importDef.Name).SetSourceType(lsv1alpha1.ImportDataObjectSourceType).
		SetData(importData)
	raw, err := do.Build()
	if err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
		o.Inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(o.Inst.Info.Status.LastError,
			"CreateDataObjects", fmt.Sprintf("unable to create dataobjects for import '%s'", importDef.Name),
			err.Error())
		return fmt.Errorf("unable to build data object for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), raw, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.Info, raw, kubernetes.LandscaperScheme); err != nil {
			return err
		}
		return do.Apply(raw)
	}); err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
		o.Inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(o.Inst.Info.Status.LastError,
			"CreateDatatObjects", fmt.Sprintf("unable to create data objects for import '%s'", importDef.Name),
			err.Error())
		return fmt.Errorf("unable to create or update data object '%s' for import '%s': %w", raw.Name, importDef.Name, err)
	}
	return nil
}

func (o *Operation) createOrUpdateTargetImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, values interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.Info.Status.Conditions, lsv1alpha1.CreateImportsCondition)
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	target := &lsv1alpha1.Target{}
	if _, _, err := serializer.NewCodecFactory(kubernetes.LandscaperScheme).UniversalDecoder().Decode(data, nil, target); err != nil {
		return err
	}
	intTarget, err := dataobjects.NewFromTarget(target)
	if err != nil {
		return err
	}
	intTarget.SetNamespace(o.Inst.Info.Namespace).
		SetContext(src).
		SetKey(importDef.Name).
		SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType)

	target, err = intTarget.Build()
	if err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create target for import '%s'", importDef.Name)))
		o.Inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(o.Inst.Info.Status.LastError,
			"CreateTargets", fmt.Sprintf("unable to create target for import '%s'", importDef.Name),
			err.Error())
		return fmt.Errorf("unable to build target for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), target, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.Info, target, kubernetes.LandscaperScheme); err != nil {
			return err
		}
		return intTarget.Apply(target)
	}); err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create target for import '%s'", importDef.Name)))
		o.Inst.Info.Status.LastError = lsv1alpha1helper.UpdatedError(o.Inst.Info.Status.LastError,
			"CreateTargets", fmt.Sprintf("unable to create target for import '%s'", importDef.Name),
			err.Error())
		return fmt.Errorf("unable to create or update target '%s' for import '%s': %w", target.Name, importDef.Name, err)
	}

	return nil
}

// GetExportForKey creates a dataobject from a dataobject
func (o *Operation) GetExportForKey(ctx context.Context, key string) (*dataobjects.DataObject, error) {
	doName := lsv1alpha1helper.GenerateDataObjectName(o.context.Name, key)
	rawDO := &lsv1alpha1.DataObject{}
	if err := o.Client().Get(ctx, kutil.ObjectKey(doName, o.Inst.Info.Namespace), rawDO); err != nil {
		return nil, err
	}
	return dataobjects.NewFromDataObject(rawDO)
}

func importsAnyExport(exporter, importer *Installation) bool {
	for _, export := range exporter.Info.Spec.Exports.Data {
		for _, def := range importer.Info.Spec.Imports.Data {
			if def.DataRef == export.DataRef {
				return true
			}
		}
	}
	for _, export := range exporter.Info.Spec.Exports.Targets {
		for _, def := range importer.Info.Spec.Imports.Targets {
			if def.Target == export.Target {
				return true
			}
		}
	}
	return false
}
