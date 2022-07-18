// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

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

	Inst                            *Installation
	ComponentDescriptor             *cdv2.ComponentDescriptor
	Overwriter                      componentoverwrites.Overwriter
	BlobResolver                    ctf.BlobResolver
	ResolvedComponentDescriptorList *cdv2.ComponentDescriptorList
	context                         Context

	targetLists map[string]*dataobjects.TargetList
	targets     map[string]*dataobjects.Target

	// CurrentOperation is the name of the current operation that is used for the error erporting
	CurrentOperation string
}

// NewInstallationOperation creates a new installation operation.
// DEPRECATED: use the builder instead.
func NewInstallationOperation(ctx context.Context, log logr.Logger, c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, cRegistry ctf.ComponentResolver, inst *Installation) (*Operation, error) {
	return NewOperationBuilder(inst).
		WithLogger(log).
		Client(c).
		Scheme(scheme).
		WithEventRecorder(recorder).
		ComponentRegistry(cRegistry).
		Build(ctx)
}

// NewInstallationOperationFromOperation creates a new installation operation from an existing common operation.
// DEPRECATED: use the builder instead.
func NewInstallationOperationFromOperation(ctx context.Context, op *lsoperation.Operation, inst *Installation, _ *cdv2.UnstructuredTypedObject) (*Operation, error) {
	return NewOperationBuilder(inst).
		WithOperation(op).
		Build(ctx)
}

func (o *Operation) GetTargetImport(name string) *dataobjects.Target {
	return o.targets[name]
}
func (o *Operation) GetTargetListImport(name string) *dataobjects.TargetList {
	return o.targetLists[name]
}
func (o *Operation) SetTargetImports(data map[string]*dataobjects.Target) {
	o.targets = data
}
func (o *Operation) SetTargetListImports(data map[string]*dataobjects.TargetList) {
	o.targetLists = data
}

// ResolveComponentDescriptors resolves the effective component descriptors for the installation.
// DEPRECATED: only used for tests. use the builder methods instead.
func (o *Operation) ResolveComponentDescriptors(ctx context.Context) error {
	cd, blobResolver, err := ResolveComponentDescriptor(ctx, o.ComponentsRegistry(), o.Inst.Info)
	if err != nil {
		return err
	}
	if cd == nil {
		return nil
	}

	resolvedCD, err := cdutils.ResolveToComponentDescriptorList(ctx, o.ComponentsRegistry(), *cd, o.Context().External.RepositoryContext)
	if err != nil {
		return err
	}
	o.ComponentDescriptor = cd
	o.BlobResolver = blobResolver
	o.ResolvedComponentDescriptorList = &resolvedCD
	return nil
}

// Log returns a modified logger for the installation.
func (o *Operation) Log() logr.Logger {
	return o.Operation.Log().WithValues("installation", types.NamespacedName{
		Namespace: o.Inst.Info.Namespace,
		Name:      o.Inst.Info.Name,
	})
}

// Context returns the context of the operated installation
func (o *Operation) Context() *Context {
	return &o.context
}

// InstallationContextName returns the name of the current installation context.
func (o *Operation) InstallationContextName() string {
	return o.context.Name
}

// JSONSchemaValidator returns a jsonschema validator.
func (o *Operation) JSONSchemaValidator(schema []byte) (*jsonschema.Validator, error) {
	v := jsonschema.NewValidator(&jsonschema.ReferenceContext{
		LocalTypes:          o.Inst.Blueprint.Info.LocalTypes,
		BlueprintFs:         o.Inst.Blueprint.Fs,
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

// SetOverwriter sets the component overwriter.
func (o *Operation) SetOverwriter(ow componentoverwrites.Overwriter) {
	o.Overwriter = ow
}

// ListSubinstallations returns a list of all subinstallations of the given installation.
// Returns nil if no installations can be found
func (o *Operation) ListSubinstallations(ctx context.Context, filter ...FilterInstallationFunc) ([]*lsv1alpha1.Installation, error) {
	return ListSubinstallations(ctx, o.Client(), o.Inst.Info, filter...)
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
func (o *Operation) UpdateInstallationStatus(ctx context.Context, inst *lsv1alpha1.Installation, phase lsv1alpha1.ComponentInstallationPhase, updatedConditions ...lsv1alpha1.Condition) error {
	inst.Status.Phase = phase
	inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, updatedConditions...)
	if err := o.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000018, inst); err != nil {
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

		do, _, err := GetDataImport(ctx, o.Client(), o.Context().Name, &o.Inst.InstallationBase, def)
		if err != nil {
			return nil, err
		}
		dataObjects[def.Name] = do

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = dataobjects.ImportedBase(do).ComputeConfigGeneration()
			owner     = kutil.GetOwner(do.Raw.ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.Info) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.Info.Namespace,
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
		if len(def.Target) == 0 {
			// It's a target list, skip it
			continue
		}
		target, err := GetTargetImport(ctx, o.Client(), o.Context().Name, o.Inst.Info, def)
		if err != nil {
			return nil, err
		}
		targets[def.Name] = target

		var (
			sourceRef *lsv1alpha1.ObjectReference
			configGen = dataobjects.ImportedBase(target).ComputeConfigGeneration()
			owner     = kutil.GetOwner(target.Raw.ObjectMeta)
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.Info) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.Info.Namespace,
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
func (o *Operation) GetImportedTargetLists(ctx context.Context) (map[string]*dataobjects.TargetList, error) {
	targets := map[string]*dataobjects.TargetList{}
	for _, def := range o.Inst.Info.Spec.Imports.Targets {
		if len(def.Target) != 0 {
			// It's a single target, skip it
			continue
		}
		var (
			tl  *dataobjects.TargetList
			err error
		)
		if def.Targets != nil {
			// List of target names
			tl, err = GetTargetListImportByNames(ctx, o.Client(), o.Context().Name, o.Inst.Info, def)
		} else if len(def.TargetListReference) != 0 {
			// TargetListReference is converted to a label selector internally
			tl, err = GetTargetListImportBySelector(ctx, o.Client(), o.Context().Name, o.Inst.Info, map[string]string{lsv1alpha1.DataObjectKeyLabel: def.TargetListReference}, def, true)
		} else {
			// Invalid target
			err = fmt.Errorf("invalid target definition '%s': none of target, targets and targetListRef is defined", def.Name)
		}
		if err != nil {
			return nil, err
		}

		targets[def.Name] = tl

		tis := make([]lsv1alpha1.TargetImportStatus, len(tl.Targets))
		for i, t := range tl.Targets {
			var (
				sourceRef *lsv1alpha1.ObjectReference
				configGen = dataobjects.ImportedBase(t).ComputeConfigGeneration()
				owner     = kutil.GetOwner(t.Raw.ObjectMeta)
			)
			if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.Info) {
				sourceRef = &lsv1alpha1.ObjectReference{
					Name:      owner.Name,
					Namespace: o.Inst.Info.Namespace,
				}
				inst := &lsv1alpha1.Installation{}
				if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
					return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
						sourceRef.NamespacedName().String(), def.Name, err)
				}
				configGen = inst.Status.ConfigGeneration
			}
			tis[i] = lsv1alpha1.TargetImportStatus{
				Target:           t.Raw.Name,
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

// GetImportedComponentDescriptors returns all imported component descriptors of the installation.
func (o *Operation) GetImportedComponentDescriptors(ctx context.Context) (map[string]*dataobjects.ComponentDescriptor, error) {
	cds := map[string]*dataobjects.ComponentDescriptor{}
	for _, def := range o.Inst.Info.Spec.Imports.ComponentDescriptors {
		if def.List != nil {
			// It's a component descriptor list, skip it
			continue
		}
		cd, err := GetComponentDescriptorImport(ctx, o.Client(), o.Context().Name, o, def)
		if err != nil {
			return nil, err
		}
		cds[def.Name] = cd

		var (
			sref, cmref string
			cdref       *lsv1alpha1.ComponentDescriptorReference = nil
			sourceRef   *lsv1alpha1.ObjectReference
			configGen   = cd.Descriptor.Version
			owner       = cd.Owner
		)
		if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.Info) {
			sourceRef = &lsv1alpha1.ObjectReference{
				Name:      owner.Name,
				Namespace: o.Inst.Info.Namespace,
			}
			inst := &lsv1alpha1.Installation{}
			if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
				return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
					sourceRef.NamespacedName().String(), def.Name, err)
			}
		}
		switch cd.RefType {
		case dataobjects.RegistryReference:
			cdref = cd.RegistryRef
		case dataobjects.ConfigMapReference:
			cmref = fmt.Sprintf("%s#%s", def.ConfigMapRef.NamespacedName().String(), def.ConfigMapRef.Key)
		case dataobjects.SecretReference:
			sref = fmt.Sprintf("%s#%s", def.SecretRef.NamespacedName().String(), def.SecretRef.Key)
		}
		o.Inst.ImportStatus().Update(lsv1alpha1.ImportStatus{
			Name:                   def.Name,
			Type:                   lsv1alpha1.CDImportStatusType,
			SecretRef:              sref,
			ConfigMapRef:           cmref,
			ComponentDescriptorRef: cdref,
			SourceRef:              sourceRef,
			ConfigGeneration:       configGen,
		})
	}

	return cds, nil
}

// GetImportedComponentDescriptorLists returns all imported component descriptor lists of the installation.
func (o *Operation) GetImportedComponentDescriptorLists(ctx context.Context) (map[string]*dataobjects.ComponentDescriptorList, error) {
	cdls := map[string]*dataobjects.ComponentDescriptorList{}
	for _, def := range o.Inst.Info.Spec.Imports.ComponentDescriptors {
		if def.List == nil {
			// It's a single component descriptor import, skip it
			continue
		}
		cdl, err := GetComponentDescriptorListImport(ctx, o.Client(), o.Context().Name, o, def)
		if err != nil {
			return nil, err
		}
		cdls[def.Name] = cdl

		cdis := make([]lsv1alpha1.CDImportStatus, len(cdl.ComponentDescriptors))
		for i, cd := range cdl.ComponentDescriptors {
			var (
				sourceRef *lsv1alpha1.ObjectReference
				owner     = cd.Owner
			)
			if OwnerReferenceIsInstallationButNoParent(owner, o.Inst.Info) {
				sourceRef = &lsv1alpha1.ObjectReference{
					Name:      owner.Name,
					Namespace: o.Inst.Info.Namespace,
				}
				inst := &lsv1alpha1.Installation{}
				if err := read_write_layer.GetInstallation(ctx, o.Client(), sourceRef.NamespacedName(), inst); err != nil {
					return nil, fmt.Errorf("unable to get source installation '%s' for import '%s': %w",
						sourceRef.NamespacedName().String(), def.Name, err)
				}
			}
			var (
				sref, cmref string
				cdref       *lsv1alpha1.ComponentDescriptorReference = nil
			)
			switch cd.RefType {
			case dataobjects.RegistryReference:
				cdref = cd.RegistryRef
			case dataobjects.ConfigMapReference:
				cmref = fmt.Sprintf("%s#%s", cd.ConfigMapRef.NamespacedName().String(), cd.ConfigMapRef.Key)
			case dataobjects.SecretReference:
				sref = fmt.Sprintf("%s#%s", cd.SecretRef.NamespacedName().String(), cd.SecretRef.Key)
			}
			cdis[i] = lsv1alpha1.CDImportStatus{
				ComponentDescriptorRef: cdref,
				ConfigMapRef:           cmref,
				SecretRef:              sref,
				SourceRef:              sourceRef,
			}
		}
		o.Inst.ImportStatus().Update(lsv1alpha1.ImportStatus{
			Name:                 def.Name,
			Type:                 lsv1alpha1.CDListImportStatusType,
			ComponentDescriptors: cdis,
		})
	}
	return cdls, nil
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

// TriggerDependents triggers all installations that depend on the current installation.
// These are most likely all installation that import a key which is exported by the current installation.
func (o *Operation) TriggerDependents(ctx context.Context) error {
	for _, sibling := range o.Context().Siblings {
		if !importsAnyExport(o.Inst, sibling) {
			continue
		}

		// todo: maybe use patch
		metav1.SetMetaDataAnnotation(&sibling.Info.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		if err := o.Writer().UpdateInstallation(ctx, read_write_layer.W000011, sibling.Info); err != nil {
			return errors.Wrapf(err, "unable to trigger installation %s", sibling.Info.Name)
		}
	}
	return nil
}

// NewTriggerDependents triggers all installations that depend on the current installation.
func (o *Operation) NewTriggerDependents(ctx context.Context) error {
	for _, sibling := range o.Context().Siblings {
		if !importsAnyExport(o.Inst, sibling) {
			continue
		}

		if IsRootInstallation(o.Inst.Info) {
			metav1.SetMetaDataAnnotation(&sibling.Info.ObjectMeta, lsv1alpha1.OperationAnnotation, string(lsv1alpha1.ReconcileOperation))
		} else {
			lsv1alpha1helper.Touch(&sibling.Info.ObjectMeta)
		}

		if err := o.Writer().UpdateInstallation(ctx, read_write_layer.W000085, sibling.Info); err != nil {
			return err
		}
	}
	return nil
}

// SetExportConfigGeneration returns the new export generation of the installation
// based on its own generation and its context
func (o *Operation) SetExportConfigGeneration(ctx context.Context) error {
	// we have to set our config generation to the desired state

	o.Inst.Info.Status.ConfigGeneration = ""
	return o.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000016, o.Inst.Info)
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
		if _, err := o.Writer().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000068, raw, func() error {
			if err, err2 := lsutil.SetExclusiveOwnerReference(o.Inst.Info, raw); err != nil {
				return fmt.Errorf("dataobject '%s' for export '%s' conflicts with existing dataobject owned by another installation: %w", client.ObjectKeyFromObject(raw).String(), do.Metadata.Key, err)
			} else if err2 != nil {
				return fmt.Errorf("error setting owner reference: %w", err2)
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
		if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000069, raw, func() error {
			if err, err2 := lsutil.SetExclusiveOwnerReference(o.Inst.Info, raw); err != nil {
				return fmt.Errorf("target object '%s' for export '%s' conflicts with existing target owned by another installation: %w", client.ObjectKeyFromObject(raw).String(), target.Metadata.Key, err)
			} else if err2 != nil {
				return fmt.Errorf("error setting owner reference: %w", err2)
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
func (o *Operation) CreateOrUpdateImports(ctx context.Context) error {
	return o.createOrUpdateImports(ctx, o.Inst.Blueprint.Info.Imports)
}

func (o *Operation) createOrUpdateImports(ctx context.Context, importDefs lsv1alpha1.ImportDefinitionList) error {
	importedValues := o.Inst.GetImports()
	src := lsv1alpha1helper.DataObjectSourceFromInstallation(o.Inst.Info)
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
		case lsv1alpha1.ImportTypeComponentDescriptor:
			// nothing to do for component descriptors, since they do not use in-cluster objects for import propagation
		case lsv1alpha1.ImportTypeComponentDescriptorList:
			// nothing to do for component descriptor lists, since they do not use in-cluster objects for import propagation
		default:
			return fmt.Errorf("unknown import type '%s' for import '%s'", string(importDef.Type), importDef.Name)
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
		return fmt.Errorf("unable to build data object for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.Writer().CreateOrUpdateCoreDataObject(ctx, read_write_layer.W000070, raw, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.Info, raw, api.LandscaperScheme); err != nil {
			return err
		}
		return do.Apply(raw)
	}); err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateDataObjects",
				fmt.Sprintf("unable to create data object for import '%s'", importDef.Name)))
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
	if _, _, err := api.Decoder.Decode(data, nil, target); err != nil {
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
		return fmt.Errorf("unable to build target for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000071, target, func() error {
		if err := controllerutil.SetOwnerReference(o.Inst.Info, target, api.LandscaperScheme); err != nil {
			return err
		}
		return intTarget.Apply(target)
	}); err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create target for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to create or update target '%s' for import '%s': %w", target.Name, importDef.Name, err)
	}

	return nil
}

func (o *Operation) createOrUpdateTargetListImport(ctx context.Context, src string, importDef lsv1alpha1.ImportDefinition, values []interface{}) error {
	cond := lsv1alpha1helper.GetOrInitCondition(o.Inst.Info.Status.Conditions, lsv1alpha1.CreateImportsCondition)
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
	intTL, err := dataobjects.NewFromTargetList(tars)
	if err != nil {
		return err
	}
	for i := range intTL.Targets {
		tar := intTL.Targets[i]
		tar.SetNamespace(o.Inst.Info.Namespace).
			SetContext(src).
			SetKey(importDef.Name).
			SetSource(src).SetSourceType(lsv1alpha1.ImportDataObjectSourceType)
	}

	targets, err := intTL.Build(importDef.Name)
	if err != nil {
		o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
			lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"CreateTargets",
				fmt.Sprintf("unable to create targets for import '%s'", importDef.Name)))
		return fmt.Errorf("unable to build targets for import '%s': %w", importDef.Name, err)
	}

	// we do not need to set controller ownership as we anyway need a separate garbage collection.
	for i, target := range targets {
		if _, err := o.Writer().CreateOrUpdateCoreTarget(ctx, read_write_layer.W000072, target, func() error {
			if err := controllerutil.SetOwnerReference(o.Inst.Info, target, api.LandscaperScheme); err != nil {
				return err
			}
			return intTL.Apply(target, i)
		}); err != nil {
			o.Inst.Info.Status.Conditions = lsv1alpha1helper.MergeConditions(o.Inst.Info.Status.Conditions,
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
	if err := o.Client().Get(ctx, kutil.ObjectKey(doName, o.Inst.Info.Namespace), rawDO); err != nil {
		return nil, err
	}
	return dataobjects.NewFromDataObject(rawDO)
}

func importsAnyExport(exporter *Installation, importer *InstallationBase) bool {
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
