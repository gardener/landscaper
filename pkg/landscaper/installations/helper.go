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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	lscheme "github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
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
func CreateInternalInstallations(ctx context.Context, compResolver ctf.ComponentResolver, installations ...*lsv1alpha1.Installation) ([]*Installation, error) {
	internalInstallations := make([]*Installation, len(installations))
	for i, inst := range installations {
		inInst, err := CreateInternalInstallation(ctx, compResolver, inst)
		if err != nil {
			return nil, err
		}
		internalInstallations[i] = inInst
	}
	return internalInstallations, nil
}

// CreateInternalInstallationBases creates internal installation bases for a list of ComponentInstallations
func CreateInternalInstallationBases(installations ...*lsv1alpha1.Installation) ([]*InstallationBase, error) {
	internalInstallations := make([]*InstallationBase, len(installations))
	for i, inst := range installations {
		inInst := CreateInternalInstallationBase(inst)
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
		repoCtx *cdv2.UnstructuredTypedObject
		ref     cdv2.ObjectMeta
	)
	//case inline component descriptor
	if inst.Spec.ComponentDescriptor.Inline != nil {
		repoCtx = inst.Spec.ComponentDescriptor.Inline.GetEffectiveRepositoryContext()
		ref = inst.Spec.ComponentDescriptor.Inline.ObjectMeta
	}
	// case remote reference
	if inst.Spec.ComponentDescriptor.Reference != nil {
		repoCtx = inst.Spec.ComponentDescriptor.Reference.RepositoryContext
		ref = inst.Spec.ComponentDescriptor.Reference.ObjectMeta()
	}
	return compRepo.ResolveWithBlobResolver(ctx, repoCtx, ref.GetName(), ref.GetVersion())
}

// CreateInternalInstallation creates an internal installation for an Installation
func CreateInternalInstallation(ctx context.Context, compResolver ctf.ComponentResolver, inst *lsv1alpha1.Installation) (*Installation, error) {
	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	blue, err := blueprints.Resolve(ctx, compResolver, cdRef, inst.Spec.Blueprint)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}

// CreateInternalInstallationBase creates an internal installation base for an Installation
func CreateInternalInstallationBase(inst *lsv1alpha1.Installation) *InstallationBase {
	return NewInstallationBase(inst)
}

// GetDataImport fetches the data import from the cluster.
func GetDataImport(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *InstallationBase,
	dataImport lsv1alpha1.DataImport) (*dataobjects.DataObject, *v1.OwnerReference, error) {

	var rawDataObject *lsv1alpha1.DataObject
	// get deploy item from current context
	if len(dataImport.DataRef) != 0 {
		rawDataObject = &lsv1alpha1.DataObject{}
		doName := helper.GenerateDataObjectName(contextName, dataImport.DataRef)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(doName, inst.Info.Namespace), rawDataObject); err != nil {
			return nil, nil, fmt.Errorf("unable to fetch data object %s (%s/%s): %w", doName, contextName, dataImport.DataRef, err)
		}
	}
	if dataImport.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := kubeClient.Get(ctx, dataImport.SecretRef.NamespacedName(), secret); err != nil {
			return nil, nil, err
		}

		rawDataObject = &lsv1alpha1.DataObject{}
		if len(dataImport.SecretRef.Key) != 0 {
			data, ok := secret.Data[dataImport.SecretRef.Key]
			if !ok {
				return nil, nil, fmt.Errorf("key %s in %s does not exist", dataImport.SecretRef.Key, dataImport.SecretRef.NamespacedName().String())
			}
			rawDataObject.Data.RawMessage = data
		} else {
			// use the whole secret as map
			data, err := json.Marshal(ByteMapToRawMessageMap(secret.Data))
			if err != nil {
				return nil, nil, fmt.Errorf("unable to marshal secret data as map: %w", err)
			}
			rawDataObject.Data.RawMessage = data
		}

		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(secret.Generation)
	}
	if dataImport.ConfigMapRef != nil {
		cm := &corev1.ConfigMap{}
		if err := kubeClient.Get(ctx, dataImport.ConfigMapRef.NamespacedName(), cm); err != nil {
			return nil, nil, err
		}

		rawDataObject = &lsv1alpha1.DataObject{}
		if cm.Data != nil {
			if len(dataImport.ConfigMapRef.Key) != 0 {
				data, ok := cm.Data[dataImport.ConfigMapRef.Key]
				if !ok {
					return nil, nil, fmt.Errorf("key %s in %s does not exist", dataImport.ConfigMapRef.Key, dataImport.ConfigMapRef.NamespacedName().String())
				}
				rawDataObject.Data.RawMessage = []byte(data)
			} else {
				// use whole configmap as json object
				data, err := json.Marshal(StringMapToRawMessageMap(cm.Data))
				if err != nil {
					return nil, nil, fmt.Errorf("unable to marshal configmap data as map: %w", err)
				}
				rawDataObject.Data.RawMessage = data
			}
		} else if cm.BinaryData != nil {
			if len(dataImport.ConfigMapRef.Key) != 0 {
				data, ok := cm.BinaryData[dataImport.ConfigMapRef.Key]
				if !ok {
					return nil, nil, fmt.Errorf("key %s in %s does not exist", dataImport.ConfigMapRef.Key, dataImport.ConfigMapRef.NamespacedName().String())
				}
				rawDataObject.Data.RawMessage = data
			} else {
				// use whole configmap as json object
				data, err := json.Marshal(ByteMapToRawMessageMap(cm.BinaryData))
				if err != nil {
					return nil, nil, fmt.Errorf("unable to marshal configmap data as map: %w", err)
				}
				rawDataObject.Data.RawMessage = data
			}
		} else {
			return nil, nil, fmt.Errorf("no data defined in the referenced configmap %s", cm.Name)
		}

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
func GetTargetImport(ctx context.Context, kubeClient client.Client, contextName string, inst *Installation, targetName string) (*dataobjects.Target, error) {
	// get deploy item from current context
	raw := &lsv1alpha1.Target{}
	targetName = helper.GenerateDataObjectName(contextName, targetName)
	if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Info.Namespace), raw); err != nil {
		return nil, err
	}

	target, err := dataobjects.NewFromTarget(raw)
	if err != nil {
		return nil, fmt.Errorf("unable to create internal target for %s: %w", targetName, err)
	}
	return target, nil
}

// GetTargetListImportByNames fetches the target imports from the cluster, based on a list of target names.
func GetTargetListImportByNames(ctx context.Context, kubeClient client.Client, contextName string, inst *Installation, targetNames []string) (*dataobjects.TargetList, error) {
	targets := make([]lsv1alpha1.Target, len(targetNames))
	for i, targetName := range targetNames {
		// get deploy item from current context
		raw := &lsv1alpha1.Target{}
		targetName = helper.GenerateDataObjectName(contextName, targetName)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Info.Namespace), raw); err != nil {
			return nil, err
		}
		targets[i] = *raw
	}
	targetList, err := dataobjects.NewFromTargetList(targets)
	if err != nil {
		return nil, err
	}

	return targetList, nil
}

// GetTargetListImportBySelector fetches the target imports from the cluster, based on a label selector.
// If restrictToImport is true, a label selector will be added which fetches only targets that are marked as import.
func GetTargetListImportBySelector(ctx context.Context, kubeClient client.Client, contextName string, inst *Installation, selector map[string]string, restrictToImport bool) (*dataobjects.TargetList, error) {
	targets := &lsv1alpha1.TargetList{}
	// construct label selector
	contextSelector := labels.NewSelector()
	if len(contextName) != 0 {
		// top-level targets probably don't have an empty context set, so only add the selector if there actually is a context
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectContextLabel, selection.Equals, []string{contextName})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	} else {
		// top-level targets probably don't have an empty context set, so check for non-existence of the label
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectContextLabel, selection.DoesNotExist, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	// add given labels to selector
	for k, v := range selector {
		r, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	if restrictToImport {
		// add further labels to ensure that only targets imported by that installation are selected
		r, err := labels.NewRequirement(lsv1alpha1.DataObjectSourceTypeLabel, selection.Equals, []string{string(lsv1alpha1.ImportDataObjectSourceType)})
		if err != nil {
			return nil, fmt.Errorf("unable to construct label selector: %w", err)
		}
		contextSelector = contextSelector.Add(*r)
	}
	if err := kubeClient.List(ctx, targets, client.InNamespace(inst.Info.Namespace), &client.ListOptions{LabelSelector: contextSelector}); err != nil {
		return nil, err
	}
	targetList, err := dataobjects.NewFromTargetList(targets.Items)
	if err != nil {
		return nil, err
	}
	return targetList, nil
}

// GetReferenceFromComponentDescriptorDefinition tries to extract a component descriptor reference from a given component descriptor definition
func GetReferenceFromComponentDescriptorDefinition(cdDef *lsv1alpha1.ComponentDescriptorDefinition) *lsv1alpha1.ComponentDescriptorReference {
	if cdDef == nil {
		return nil
	}

	if cdDef.Inline != nil {
		repoCtx := cdDef.Inline.GetEffectiveRepositoryContext()
		return &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: repoCtx,
			ComponentName:     cdDef.Inline.Name,
			Version:           cdDef.Inline.Version,
		}
	}

	return cdDef.Reference
}

// ByteMapToRawMessageMap converts a map of bytes to a map of json.RawMessages
func ByteMapToRawMessageMap(m map[string][]byte) map[string]json.RawMessage {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		n[key] = json.RawMessage(val)
	}
	return n
}

// StringMapToRawMessageMap converts a map of strings to a map of json.RawMessages
func StringMapToRawMessageMap(m map[string]string) map[string]json.RawMessage {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		n[key] = json.RawMessage(val)
	}
	return n
}
