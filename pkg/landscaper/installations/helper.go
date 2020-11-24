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

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1/helper"
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
func ResolveComponentDescriptor(ctx context.Context, compRepo ctf.ComponentResolver, inst *lsv1alpha1.Installation) (*cdv2.ComponentDescriptor, ctf.BlobResolver, error) {
	if inst.Spec.Blueprint.Reference == nil &&
		(inst.Spec.Blueprint.Inline == nil || inst.Spec.Blueprint.Inline.ComponentDescriptorReference == nil) {
		return nil, nil, nil
	}
	var (
		repoCtx cdv2.RepositoryContext
		ref     cdv2.ObjectMeta
	)
	if inst.Spec.Blueprint.Reference != nil {
		// todo: if not defined read from default configured repo context.
		repoCtx = *inst.Spec.Blueprint.Reference.RepositoryContext
		ref = inst.Spec.Blueprint.Reference.ObjectMeta()
	}
	if inst.Spec.Blueprint.Inline != nil && inst.Spec.Blueprint.Inline.ComponentDescriptorReference != nil {
		repoCtx = *inst.Spec.Blueprint.Inline.ComponentDescriptorReference.RepositoryContext
		ref = inst.Spec.Blueprint.Inline.ComponentDescriptorReference.ObjectMeta()
	}
	return compRepo.Resolve(ctx, repoCtx, ref.GetName(), ref.GetVersion())
}

// CreateInternalInstallation creates an internal installation for a Installation
func CreateInternalInstallation(ctx context.Context, op lsoperation.Interface, inst *lsv1alpha1.Installation) (*Installation, error) {
	blue, err := blueprints.Resolve(ctx, op.ComponentsRegistry(), inst.Spec.Blueprint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}

// GetDataImport fetches the data import from the cluster.
func GetDataImport(ctx context.Context, op lsoperation.Interface, contextName string, inst *Installation, dataRef lsv1alpha1.DataImport) (*dataobjects.DataObject, *v1.OwnerReference, error) {
	var rawDataObject *lsv1alpha1.DataObject
	// get deploy item from current context
	if len(dataRef.DataRef) != 0 {
		rawDataObject = &lsv1alpha1.DataObject{}
		doName := helper.GenerateDataObjectName(contextName, dataRef.DataRef)
		if err := op.Client().Get(ctx, kubernetes.ObjectKey(doName, inst.Info.Namespace), rawDataObject); err != nil {
			return nil, nil, err
		}
	}
	if dataRef.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := op.Client().Get(ctx, dataRef.SecretRef.NamespacedName(), secret); err != nil {
			return nil, nil, err
		}
		data, ok := secret.Data[dataRef.SecretRef.Key]
		if !ok {
			return nil, nil, fmt.Errorf("key %s in %s does not exist", dataRef.SecretRef.Key, dataRef.SecretRef.NamespacedName().String())
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(secret.Generation)
	}
	if dataRef.ConfigMapRef != nil {
		cm := &corev1.ConfigMap{}
		if err := op.Client().Get(ctx, dataRef.ConfigMapRef.NamespacedName(), cm); err != nil {
			return nil, nil, err
		}
		data, ok := cm.Data[dataRef.ConfigMapRef.Key]
		if !ok {
			return nil, nil, fmt.Errorf("key %s in %s does not exist", dataRef.SecretRef.Key, dataRef.SecretRef.NamespacedName().String())
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data = []byte(data)
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
