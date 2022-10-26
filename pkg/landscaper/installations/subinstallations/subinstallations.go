// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package subinstallations

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/pkg/utils/dependencies"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

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

// Ensure ensures that all referenced definitions are mapped to a sub-installation.
func (o *Operation) Ensure(ctx context.Context) error {
	var (
		inst = o.Inst.GetInstallation()
		cond = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
	)

	o.CurrentOperation = string(lsv1alpha1.EnsureSubInstallationsCondition)

	subInstallations, err := o.GetSubInstallations(ctx, inst)
	if err != nil {
		return err
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
			_, ok := o.Inst.GetImports()[imp.DataRef]
			if ok || !isOptionalParentImport(imp.DataRef, o.Inst.GetBlueprint().Info.Imports, false) {
				imports = append(imports, imp)
			}
		}
		instT.Imports.Data = imports
	}

	// validate all installation templates before do any follow up actions
	if err := o.ValidateSubinstallations(installationTmpl); err != nil {
		return err
	}

	// delete removed subreferences
	_, err = o.cleanupOrphanedSubInstallations(ctx, subInstallations, installationTmpl)
	if err != nil {
		return err
	}

	if err := o.createOrUpdateSubinstallations(ctx, subInstallations, installationTmpl); err != nil {
		return err
	}

	cond = lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionTrue,
		"InstallationsInstalled", "All Installations are successfully installed")
	return o.UpdateInstallationStatus(ctx, inst, cond)
}

// isOptionalParentImport returns true if the specified import data reference
// - exists in the parents blueprint (= in the given import definition list) AND
//   - is optional (required: false) OR
//   - is a conditional import
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
			name, ok = getSubinstallationNameByReference(o.Inst.GetInstallation().Status.InstallationReferences, inst.Namespace, inst.Name)
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

	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{lc.KeyReconciledResource, client.ObjectKeyFromObject(o.Inst.GetInstallation()).String()})

	var (
		inst    = o.Inst.GetInstallation()
		cond    = lsv1alpha1helper.GetOrInitCondition(inst.Status.Conditions, lsv1alpha1.EnsureSubInstallationsCondition)
		deleted = false
	)

	for defName, subInst := range subInstallations {
		if _, ok := getInstallationTemplate(installationTmpl, defName); ok {
			continue
		}

		// delete installation
		logger.Info("delete orphaned installation", "name", subInst.Name)

		metav1.SetMetaDataAnnotation(&subInst.ObjectMeta, lsv1alpha1.DeleteIgnoreSuccessors, "true")

		if err := o.Writer().UpdateInstallation(ctx, read_write_layer.W000015, subInst); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			return deleted, o.NewError(err, "UpdateInstallationDeleteIgnoreSuccessors", err.Error())
		}

		if err := o.Writer().DeleteInstallation(ctx, read_write_layer.W000021, subInst); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
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
	if len(o.Inst.GetBlueprint().Info.SubinstallationExecutions) != 0 {
		templateStateHandler := template.KubernetesStateHandler{
			KubeClient: o.Client(),
			Inst:       o.Inst.GetInstallation(),
		}
		tmpl := template.New(gotemplate.New(o.BlobResolver, templateStateHandler), spiff.New(templateStateHandler))
		templatedTmpls, err := tmpl.TemplateSubinstallationExecutions(template.NewDeployExecutionOptions(
			template.NewBlueprintExecutionOptions(
				o.Context().External.InjectComponentDescriptorRef(o.Inst.GetInstallation().DeepCopy()),
				o.Inst.GetBlueprint(),
				o.ComponentDescriptor,
				o.ResolvedComponentDescriptorList,
				o.Inst.GetImports())))

		if err != nil {
			return nil, fmt.Errorf("unable to template subinstllations: %w", err)
		}
		instTmpls = append(instTmpls, templatedTmpls...)
	}
	if len(o.Inst.GetBlueprint().Info.Subinstallations) != 0 {
		defaultTemplates, err := o.Inst.GetBlueprint().GetSubinstallations()
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

	if _, err := dependencies.CheckForCyclesAndDuplicateExports(installationTmpl, false); err != nil {
		return err
	}

	for _, subInstTmpl := range installationTmpl {
		subInst := subInstallations[subInstTmpl.Name]
		if subInst != nil && !subInst.ObjectMeta.DeletionTimestamp.IsZero() {
			// if a subinstallation was deleted, the deletion failed and it should be created again
			// in such a situation the subinstallation must be removed first
			return fmt.Errorf("an installation %s should be created which is currently under deletion", subInst.Name)
		}

		_, err := o.createOrUpdateNewInstallation(ctx, o.Inst.GetInstallation(), subInstTmpl, subInst)
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
		o.ComponentsRegistry(),
		o.Context().External.RepositoryContext,
		o.Context().External.Overwriter)
	if err != nil {
		return nil, err
	}

	_, err = o.Writer().CreateOrUpdateInstallation(ctx, read_write_layer.W000001, subInst, func() error {
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
			Context:             inst.Spec.Context,
			RegistryPullSecrets: inst.Spec.RegistryPullSecrets,
			ComponentDescriptor: subCdDef,
			Blueprint:           *subBlueprint,
			Imports:             subInstTmpl.Imports,
			ImportDataMappings:  subInstTmpl.ImportDataMappings,
			Exports:             subInstTmpl.Exports,
			ExportDataMappings:  subInstTmpl.ExportDataMappings,
		}

		o.Scheme().Default(subInst)
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

	oldStatus := inst.Status.DeepCopy()
	// update or create installation reference
	var installationReference *lsv1alpha1.NamedObjectReference = nil
	for _, ref := range inst.Status.InstallationReferences {
		if ref.Name == subInstTmpl.Name {
			ref.Reference.Name = subInst.Name
			ref.Reference.Namespace = subInst.Namespace
			installationReference = &ref
			break
		}
	}

	if installationReference == nil {
		inst.Status.InstallationReferences = append(inst.Status.InstallationReferences, lsv1alpha1helper.NewInstallationReferenceState(subInstTmpl.Name, subInst))
	}

	if !reflect.DeepEqual(oldStatus, inst.Status) {
		if err := o.Writer().UpdateInstallationStatus(ctx, read_write_layer.W000017, inst); err != nil {
			return nil, errors.Wrapf(err, "unable to add new installation for %s to state", subInstTmpl.Name)
		}
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
