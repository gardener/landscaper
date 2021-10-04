// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/spiff"

	"github.com/gardener/landscaper/apis/core/validation"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
)

// TriggerSubInstallations triggers a reconcile for all sub installation of the component.
func (o *Operation) TriggerSubInstallations(ctx context.Context, inst *lsv1alpha1.Installation, operation lsv1alpha1.Operation) error {
	for _, instRef := range inst.Status.InstallationReferences {
		subInst := &lsv1alpha1.Installation{}
		if err := o.Client().Get(ctx, instRef.Reference.NamespacedName(), subInst); err != nil {
			return errors.Wrapf(err, "unable to get sub installation %s", instRef.Reference.NamespacedName().String())
		}

		metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.OperationAnnotation, string(operation))
		if err := o.Client().Update(ctx, subInst); err != nil {
			return errors.Wrapf(err, "unable to update sub installation %s", instRef.Reference.NamespacedName().String())
		}
	}
	return nil
}

// Ensure ensures that all referenced definitions are mapped to a sub-installation.
func (o *Operation) Ensure(ctx context.Context) error {
	var (
		inst = o.Inst.Info
		cond = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
	)

	o.CurrentOperation = string(lsv1alpha1.EnsureSubInstallationsCondition)

	subInstallations, err := o.GetSubInstallations(ctx, inst)
	if err != nil {
		return err
	}

	// need to check if we are allowed to update the subinstallation
	// - we are not allowed if any subresource is in deletion
	// - we are not allowed to update if any subinstallation is progressing
	for _, subInstallations := range subInstallations {
		if subInstallations.DeletionTimestamp != nil {
			inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
			err := fmt.Errorf("not eligible for update due to deletion of subinstallation %s", subInstallations.Name)
			return o.NewError(err, "DeletingSubInstallation", err.Error())
		}

		if subInstallations.Status.Phase == lsv1alpha1.ComponentPhaseProgressing {
			inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
			err = fmt.Errorf("not eligible for update due to running subinstallation %s", subInstallations.Name)
			return o.NewError(err, "RunningSubinstallation", err.Error())
		}
	}

	installationTmpl, err := o.getInstallationTemplates()
	if err != nil {
		err = fmt.Errorf("unable to get installation templates of blueprint: %w", err)
		return o.NewError(err, "GetInstallationTemplates", err.Error())
	}

	for _, instT := range installationTmpl {
		// remove imports based on optional and conditional imports which are not satisfied in the parent
		imports := []lsv1alpha1.DataImport{}
		for _, imp := range instT.Imports.Data {
			_, ok := o.Inst.Imports[imp.DataRef]
			if ok || !isOptionalParentImport(imp.DataRef, o.Inst.Blueprint.Info.Imports, false) {
				imports = append(imports, imp)
			}
		}
		instT.Imports.Data = imports
	}

	// validate all installation templates before do any follow up actions
	if err := o.ValidateSubinstallations(installationTmpl); err != nil {
		return err
	}

	for _, instT := range installationTmpl {
		// replace data references component descriptor imports with the corresponding parent import
		// this will eventually lead to a 'direct' reference
		// needs to happen after subinstallation template validation because the validation accepts only data references in subinstallation templates, which are replaced in this step
		for i := range instT.Imports.ComponentDescriptors {
			imp := &instT.Imports.ComponentDescriptors[i]
			if len(imp.DataRef) != 0 {
				cdimp, err := o.Inst.GetCDImport(imp.DataRef)
				if err != nil {
					return fmt.Errorf("unable to resolve data reference for component descriptor import '%s' in subinstallation: import '%s' not found in parent: %w", imp.Name, imp.DataRef, err)
				}
				// set import to that of the parent
				imp.DataRef = cdimp.DataRef
				imp.Ref = cdimp.Ref
				imp.ConfigMapRef = cdimp.ConfigMapRef
				imp.SecretRef = cdimp.SecretRef
				imp.List = cdimp.List
			} else if len(imp.List) != 0 {
				for j := range imp.List {
					limp := &imp.List[j]
					if len(limp.DataRef) != 0 {
						cdlimp, err := o.Inst.GetCDImport(limp.DataRef)
						if err != nil {
							return fmt.Errorf("unable to resolve data reference for component descriptor import '%s' at index %d in subinstallation: import '%s' not found in parent: %w", imp.Name, j, limp.DataRef, err)
						}
						// set import to that of the parent
						limp.DataRef = cdlimp.DataRef
						limp.Ref = cdlimp.Ref
						limp.ConfigMapRef = cdlimp.ConfigMapRef
						limp.SecretRef = cdlimp.SecretRef
					}
				}
			}
		}
	}

	// delete removed subreferences
	deletionTriggered, err := o.cleanupOrphanedSubInstallations(ctx, subInstallations, installationTmpl)
	if err != nil {
		return err
	}
	if deletionTriggered {
		return nil
	}

	if err := o.createOrUpdateSubinstallations(ctx, subInstallations, installationTmpl); err != nil {
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"InstallationsInstalled", "All Installations are successfully installed")
	return o.UpdateInstallationStatus(ctx, inst, inst.Status.Phase, cond)
}

// isOptionalParentImport returns true if the specified import data reference
// - exists in the parents blueprint (= in the given import definition list) AND
//   - is optional (required: false) OR
//	 - is a conditional import
func isOptionalParentImport(impRef string, impDefs lsv1alpha1.ImportDefinitionList, isConditional bool) bool {
	for _, imp := range impDefs {
		if imp.Name == impRef {
			return isConditional || (imp.Required != nil && !*imp.Required)
		}
		if imp.ConditionalImports != nil && len(imp.ConditionalImports) > 0 {
			if ok := isOptionalParentImport(impRef, imp.ConditionalImports, true); ok {
				return true
			}
		}
	}
	return false
}

// GetSubInstallations returns a map of all subinstallations indexed by the unique blueprint ref name.
func (o *Operation) GetSubInstallations(ctx context.Context, inst *lsv1alpha1.Installation) (map[string]*lsv1alpha1.Installation, error) {
	var (
		cond             = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
		subInstallations = map[string]*lsv1alpha1.Installation{}

		// track all found subinstallations to track if some installations were deleted
		updatedSubInstallationStates = make([]lsv1alpha1.NamedObjectReference, 0)
	)

	installations, err := o.ListSubinstallations(ctx)
	if err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"SubInstallationsNotFound", "Unable to list subinstallations")
		inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
		_ = o.CreateEventFromCondition(ctx, inst, cond)
		return nil, o.NewError(err, "SubInstallationsNotFound", err.Error())
	}
	for _, inst := range installations {
		name, ok := inst.Annotations[lsv1alpha1.SubinstallationNameAnnotation]
		if !ok {
			// todo: remove after some deprecation period.
			name, ok = getSubinstallationNameByReference(o.Inst.Info.Status.InstallationReferences, inst.Namespace, inst.Name)
			if !ok {
				err := fmt.Errorf("dangling installation found %s", inst.Name)
				return nil, o.NewError(err, "DanglingSubinstallation", err.Error())
			}
		}
		subInstallations[name] = inst
		updatedSubInstallationStates = append(updatedSubInstallationStates, lsv1alpha1.NamedObjectReference{
			Name: name,
			Reference: lsv1alpha1.ObjectReference{
				Name:      inst.Name,
				Namespace: inst.Namespace,
			},
		})
	}

	// update the sub components if installations changed
	inst.Status.InstallationReferences = updatedSubInstallationStates
	return subInstallations, nil
}

func (o *Operation) cleanupOrphanedSubInstallations(ctx context.Context,
	subInstallations map[string]*lsv1alpha1.Installation,
	installationTmpl []*lsv1alpha1.InstallationTemplate) (bool, error) {
	var (
		inst    = o.Inst.Info
		cond    = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
		deleted = false
	)

	for defName, subInst := range subInstallations {
		if _, ok := getInstallationTemplate(installationTmpl, defName); ok {
			continue
		}

		// delete installation
		o.Log().V(3).Info("delete orphaned installation", "name", subInst.Name)
		if err := o.Client().Delete(ctx, subInst); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			inst.Status.Phase = lsv1alpha1.ComponentPhaseFailed
			cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
				"InstallationNotDeleted", fmt.Sprintf("Sub Installation %s cannot be deleted", subInst.Name))
			inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
			_ = o.CreateEventFromCondition(ctx, inst, cond)
			return deleted, o.NewError(err, "InstallationNotDeleted", err.Error())
		}
		deleted = true
	}
	return deleted, nil
}

// getInstallationTemplates returns all installation templates defined by the referenced blueprint.
func (o *Operation) getInstallationTemplates() ([]*lsv1alpha1.InstallationTemplate, error) {
	var instTmpls []*lsv1alpha1.InstallationTemplate
	if len(o.Inst.Blueprint.Info.SubinstallationExecutions) != 0 {
		templateStateHandler := template.KubernetesStateHandler{
			KubeClient: o.Client(),
			Inst:       o.Inst.Info,
		}
		tmpl := template.New(gotemplate.New(o.BlobResolver, templateStateHandler), spiff.New(templateStateHandler))
		templatedTmpls, err := tmpl.TemplateSubinstallationExecutions(template.DeployExecutionOptions{
			Imports:              o.Inst.GetImports(),
			Installation:         o.Context().External.InjectComponentDescriptorRef(o.Inst.Info.DeepCopy()),
			Blueprint:            o.Inst.Blueprint,
			ComponentDescriptor:  o.ComponentDescriptor,
			ComponentDescriptors: o.ResolvedComponentDescriptorList,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to template subinstllations: %w", err)
		}
		instTmpls = append(instTmpls, templatedTmpls...)
	}
	if len(o.Inst.Blueprint.Info.Subinstallations) != 0 {
		defaultTemplates, err := o.Inst.Blueprint.GetSubinstallations()
		if err != nil {
			return nil, fmt.Errorf("unable to get default subinstallation templates: %w", err)
		}
		instTmpls = append(instTmpls, defaultTemplates...)
	}
	return instTmpls, nil
}

func (o *Operation) createOrUpdateSubinstallations(ctx context.Context,
	subInstallations map[string]*lsv1alpha1.Installation,
	installationTmpl []*lsv1alpha1.InstallationTemplate) error {
	if len(installationTmpl) == 0 {
		// do nothing
		return nil
	}

	for _, subInstTmpl := range installationTmpl {
		subInst := subInstallations[subInstTmpl.Name]
		_, err := o.createOrUpdateNewInstallation(ctx, o.Inst.Info, subInstTmpl, subInst)
		if err != nil {
			err = fmt.Errorf("unable to create installation for %s: %w", subInstTmpl.Name, err)
			return o.NewError(err, "CreateOrUpdateInstallation", err.Error())
		}
	}
	return nil
}

func (o *Operation) createOrUpdateNewInstallation(ctx context.Context,
	inst *lsv1alpha1.Installation,
	subInstTmpl *lsv1alpha1.InstallationTemplate,
	subInst *lsv1alpha1.Installation) (*lsv1alpha1.Installation, error) {
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)

	if subInst == nil {
		subInst = &lsv1alpha1.Installation{}

		generateName := subInstTmpl.Name
		if len(generateName) > validation.InstallationGenerateNameMaxLength-1 {
			generateName = generateName[:validation.InstallationGenerateNameMaxLength-1]
		}

		subInst.GenerateName = fmt.Sprintf("%s-", generateName)
		subInst.Namespace = inst.Namespace
	}

	subBlueprint, subCdDef, err := GetBlueprintDefinitionFromInstallationTemplate(inst,
		subInstTmpl,
		o.ComponentDescriptor,
		o.ComponentsRegistry())
	if err != nil {
		return nil, err
	}

	_, err = controllerruntime.CreateOrUpdate(ctx, o.Client(), subInst, func() error {
		subInst.Labels = map[string]string{
			lsv1alpha1.EncompassedByLabel: inst.Name,
		}
		subInst.Annotations = map[string]string{
			lsv1alpha1.SubinstallationNameAnnotation: subInstTmpl.Name,
		}
		if err := controllerutil.SetControllerReference(inst, subInst, o.Scheme()); err != nil {
			return errors.Wrapf(err, "unable to set owner reference")
		}
		subInst.Spec = lsv1alpha1.InstallationSpec{
			RegistryPullSecrets: inst.Spec.RegistryPullSecrets,
			ComponentDescriptor: subCdDef,
			Blueprint:           *subBlueprint,
			Imports:             subInstTmpl.Imports,
			ImportDataMappings:  subInstTmpl.ImportDataMappings,
			Exports:             subInstTmpl.Exports,
			ExportDataMappings:  subInstTmpl.ExportDataMappings,
		}

		return nil
	})
	if err != nil {
		cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			"InstallationCreatingFailed",
			fmt.Sprintf("Sub Installation %s cannot be created", subInstTmpl.Name))
		inst.Status.Conditions = lsv1alpha1helper.MergeConditions(inst.Status.Conditions, cond)
		_ = o.CreateEventFromCondition(ctx, inst, cond)
		return nil, errors.Wrapf(err, "unable to create installation for %s", subInstTmpl.Name)
	}

	// add newly created installation to state
	inst.Status.InstallationReferences = append(inst.Status.InstallationReferences, lsv1alpha1helper.NewInstallationReferenceState(subInstTmpl.Name, subInst))
	// todo: erevaluate if we really need that call
	if err := o.Client().Status().Update(ctx, inst); err != nil {
		return nil, errors.Wrapf(err, "unable to add new installation for %s to state", subInstTmpl.Name)
	}

	return subInst, nil
}

// getSubinstallationNameByReference returns the name of subinstallation by the refernce
func getSubinstallationNameByReference(refs []lsv1alpha1.NamedObjectReference, namespace, name string) (string, bool) {
	for _, ref := range refs {
		if ref.Reference.Namespace == namespace && ref.Reference.Name == name {
			return ref.Name, true
		}
	}
	return "", false
}
