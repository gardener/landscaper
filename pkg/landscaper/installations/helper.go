// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lscheme "github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils/kubernetes"
)

var componentInstallationGVK schema.GroupVersionKind

func init() {
	var err error
	componentInstallationGVK, err = apiutil.GVKForObject(&lsv1alpha1.Installation{}, lscheme.LandscaperScheme)
	runtime.Must(err)
}

// IsRootInstallation returns if the installation is a root element.
func IsRootInstallation(inst *lsv1alpha1.Installation) bool {
	_, isOwned := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return !isOwned
}

// GetParentInstallationName returns the name of parent installation that encompasses the given installation.
func GetParentInstallationName(inst *lsv1alpha1.Installation) string {
	name, _ := kubernetes.OwnerOfGVK(inst.OwnerReferences, componentInstallationGVK)
	return name
}

// CreateInternalInstallations creates internal installations for a list of ComponentInstallations
func CreateInternalInstallations(ctx context.Context, op lsoperation.Interface, installations ...*lsv1alpha1.Installation) ([]*Installation, error) {
	internalInstallations := make([]*Installation, len(installations))
	for i, inst := range installations {
		inInst, err := CreateInternalInstallation(ctx, op, inst)
		if err != nil {
			return nil, err
		}
		internalInstallations[i] = inInst
	}
	return internalInstallations, nil
}

// ResolveComponentDescriptor resolves the component descriptor of a installation.
// Inline Component Descriptors take precedence
func ResolveComponentDescriptor(ctx context.Context, compRepo ctf.ComponentResolver, inst *lsv1alpha1.Installation) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	if inst.Spec.ComponentDescriptor == nil || (inst.Spec.ComponentDescriptor.Reference == nil && inst.Spec.ComponentDescriptor.Inline == nil) {
		return nil, nil, nil
	}
	var (
		repoCtx cdv2.RepositoryContext
		ref     cdv2.ObjectMeta
	)
	//case inline component descriptor
	if inst.Spec.ComponentDescriptor.Inline != nil {
		repoCtx = inst.Spec.ComponentDescriptor.Inline.GetEffectiveRepositoryContext()
		ref = inst.Spec.ComponentDescriptor.Inline.ObjectMeta
	}
	// case remote reference
	if inst.Spec.ComponentDescriptor.Reference != nil {
		repoCtx = *inst.Spec.ComponentDescriptor.Reference.RepositoryContext
		ref = inst.Spec.ComponentDescriptor.Reference.ObjectMeta()
	}
	return compRepo.Resolve(ctx, repoCtx, ref.GetName(), ref.GetVersion())
}

// CreateInternalInstallation creates an internal installation for a Installation
func CreateInternalInstallation(ctx context.Context, op lsoperation.Interface, inst *lsv1alpha1.Installation) (*Installation, error) {
	cdRef := GeReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	blue, err := blueprints.Resolve(ctx, op.ComponentsRegistry(), cdRef, inst.Spec.Blueprint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}

// GetDataImport fetches the data import from the cluster.
func GetDataImport(ctx context.Context, op lsoperation.Interface, contextName string, inst *Installation, dataImport lsv1alpha1.DataImport) (*dataobjects.DataObject, *v1.OwnerReference, error) {
	var rawDataObject *lsv1alpha1.DataObject
	// get deploy item from current context
	if len(dataImport.DataRef) != 0 {
		rawDataObject = &lsv1alpha1.DataObject{}
		doName := helper.GenerateDataObjectName(contextName, dataImport.DataRef)
		if err := op.Client().Get(ctx, kubernetes.ObjectKey(doName, inst.Info.Namespace), rawDataObject); err != nil {
			return nil, nil, fmt.Errorf("unable to fetch data object %s (%s/%s): %w", doName, contextName, dataImport.DataRef, err)
		}
	}
	if dataImport.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := op.Client().Get(ctx, dataImport.SecretRef.NamespacedName(), secret); err != nil {
			return nil, nil, err
		}
		data, ok := secret.Data[dataImport.SecretRef.Key]
		if !ok {
			return nil, nil, fmt.Errorf("key %s in %s does not exist", dataImport.SecretRef.Key, dataImport.SecretRef.NamespacedName().String())
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(secret.Generation)
	}
	if dataImport.ConfigMapRef != nil {
		cm := &corev1.ConfigMap{}
		if err := op.Client().Get(ctx, dataImport.ConfigMapRef.NamespacedName(), cm); err != nil {
			return nil, nil, err
		}
		data, ok := cm.Data[dataImport.ConfigMapRef.Key]
		if !ok {
			return nil, nil, fmt.Errorf("key %s in %s does not exist", dataImport.SecretRef.Key, dataImport.SecretRef.NamespacedName().String())
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = []byte(data)
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(cm.Generation)
	}

	do, err := dataobjects.NewFromDataObject(rawDataObject)
	if err != nil {
		return nil, nil, err
	}

	owner := kubernetes.GetOwner(do.Raw.ObjectMeta)
	return do, owner, nil
}

// GetTargetImport fetches the target import from the cluster.
func GetTargetImport(ctx context.Context, op lsoperation.Interface, contextName string, inst *Installation, targetName string) (*dataobjects.Target, *v1.OwnerReference, error) {
	// get deploy item from current context
	raw := &lsv1alpha1.Target{}
	targetName = helper.GenerateDataObjectName(contextName, targetName)
	if err := op.Client().Get(ctx, kubernetes.ObjectKey(targetName, inst.Info.Namespace), raw); err != nil {
		return nil, nil, err
	}

	owner := kubernetes.GetOwner(raw.ObjectMeta)
	target, err := dataobjects.NewFromTarget(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create internal target for %s: %w", targetName, err)
	}
	return target, owner, nil
}

// GeReferenceFromComponentDescriptorDefinition tries to extract a component descriptor reference from a given component descriptor definition
func GeReferenceFromComponentDescriptorDefinition(cdDef *lsv1alpha1.ComponentDescriptorDefinition) *lsv1alpha1.ComponentDescriptorReference {
	if cdDef == nil {
		return nil
	}

	if cdDef.Inline != nil {
		repoCtx := cdDef.Inline.GetEffectiveRepositoryContext()
		return &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repoCtx,
			ComponentName:     cdDef.Inline.Name,
			Version:           cdDef.Inline.Version,
		}
	}

	return cdDef.Reference
}
