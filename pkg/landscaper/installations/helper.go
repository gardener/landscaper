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
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lscheme "github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
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

// CreateInternalInstallationBases creates internal installation bases for a list of ComponentInstallations
func CreateInternalInstallationBases(installations ...*lsv1alpha1.Installation) []*InstallationBase {
	if len(installations) == 0 {
		return nil
	}
	internalInstallations := make([]*InstallationBase, len(installations))
	for i, inst := range installations {
		inInst := CreateInternalInstallationBase(inst)
		internalInstallations[i] = inInst
	}
	return internalInstallations
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
// DEPRECATED: use CreateInternalInstallationWithContext instead
func CreateInternalInstallation(ctx context.Context, compResolver ctf.ComponentResolver, inst *lsv1alpha1.Installation) (*Installation, error) {
	if inst == nil {
		return nil, nil
	}
	cdRef := GetReferenceFromComponentDescriptorDefinition(inst.Spec.ComponentDescriptor)
	blue, err := blueprints.Resolve(ctx, compResolver, cdRef, inst.Spec.Blueprint)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}

// CreateInternalInstallationWithContext creates an internal installation for an Installation
func CreateInternalInstallationWithContext(ctx context.Context,
	inst *lsv1alpha1.Installation,
	kubeClient client.Client,
	compResolver ctf.ComponentResolver,
	overwriter componentoverwrites.Overwriter,
) (*Installation, error) {
	if inst == nil {
		return nil, nil
	}
	lsCtx, err := GetExternalContext(ctx, kubeClient, inst, overwriter)
	if err != nil {
		return nil, err
	}
	blue, err := blueprints.Resolve(ctx, compResolver, lsCtx.ComponentDescriptorRef(), inst.Spec.Blueprint)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve blueprint for %s/%s: %w", inst.Namespace, inst.Name, err)
	}
	return New(inst, blue)
}

// CreateInternalInstallationBase creates an internal installation base for an Installation
func CreateInternalInstallationBase(inst *lsv1alpha1.Installation) *InstallationBase {
	if inst == nil {
		return nil
	}
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
		doName := lsv1alpha1helper.GenerateDataObjectName(contextName, dataImport.DataRef)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(doName, inst.Info.Namespace), rawDataObject); err != nil {
			return nil, nil, fmt.Errorf("unable to fetch data object %s (%s/%s): %w", doName, contextName, dataImport.DataRef, err)
		}
	}
	if dataImport.SecretRef != nil {
		_, data, gen, err := ResolveSecretReference(ctx, kubeClient, dataImport.SecretRef)
		if err != nil {
			return nil, nil, err
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(gen)
	}
	if dataImport.ConfigMapRef != nil {
		_, data, gen, err := ResolveConfigMapReference(ctx, kubeClient, dataImport.ConfigMapRef)
		if err != nil {
			return nil, nil, err
		}
		rawDataObject = &lsv1alpha1.DataObject{}
		rawDataObject.Data.RawMessage = data
		// set the generation as it is used to detect outdated imports.
		rawDataObject.SetGeneration(gen)
	}

	do, err := dataobjects.NewFromDataObject(rawDataObject)
	if err != nil {
		return nil, nil, err
	}
	do.Def = &dataImport

	owner := kubernetes.GetOwner(do.Raw.ObjectMeta)
	return do, owner, nil
}

// GetTargets returns all targets and import references defined by a target import.
func GetTargets(ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	targetImport lsv1alpha1.TargetImport) ([]*dataobjects.Target, []string, error) {
	// get deploy item from current context
	var targets []*dataobjects.Target
	var targetImportReferences []string
	if len(targetImport.Target) != 0 {
		target, err := GetTargetImport(ctx, kubeClient, contextName, inst, targetImport)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get target for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		targets = []*dataobjects.Target{target}
		targetImportReferences = []string{targetImport.Target}
	} else if targetImport.Targets != nil {
		tl, err := GetTargetListImportByNames(ctx, kubeClient, contextName, inst, targetImport)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get targetlist for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		if len(tl.Targets) != len(targetImport.Targets) {
			return nil, nil, fmt.Errorf("%s: targetlist size mismatch: %d targets were expected but %d were fetched from the cluster", targetImport.Name, len(targetImport.Targets), len(tl.Targets))
		}
		targets = tl.Targets
		targetImportReferences = targetImport.Targets
	} else if len(targetImport.TargetListReference) != 0 {
		tl, err := GetTargetListImportBySelector(ctx, kubeClient, contextName, inst, map[string]string{lsv1alpha1.DataObjectKeyLabel: targetImport.TargetListReference}, targetImport, true)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: unable to get targetlist for '%s': %w", targetImport.Name, targetImport.Name, err)
		}
		targets = tl.Targets
		targetImportReferences = []string{targetImport.TargetListReference}
	} else {
		return nil, nil, fmt.Errorf("invalid target import '%s': one of target, targets, or targetListRef must be specified", targetImport.Name)
	}
	return targets, targetImportReferences, nil
}

// GetTargetImport fetches the target import from the cluster.
func GetTargetImport(ctx context.Context, kubeClient client.Client, contextName string, inst *lsv1alpha1.Installation, targetImport lsv1alpha1.TargetImport) (*dataobjects.Target, error) {
	// get deploy item from current context
	targetName := targetImport.Target
	raw := &lsv1alpha1.Target{}
	targetName = lsv1alpha1helper.GenerateDataObjectName(contextName, targetName)
	if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Namespace), raw); err != nil {
		return nil, err
	}

	target, err := dataobjects.NewFromTarget(raw)
	if err != nil {
		return nil, fmt.Errorf("unable to create internal target for %s: %w", targetName, err)
	}
	target.Def = &targetImport

	return target, nil
}

// GetTargetListImportByNames fetches the target imports from the cluster, based on a list of target names.
func GetTargetListImportByNames(
	ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	targetImport lsv1alpha1.TargetImport) (*dataobjects.TargetList, error) {
	targets := make([]lsv1alpha1.Target, len(targetImport.Targets))
	for i, targetName := range targetImport.Targets {
		// get deploy item from current context
		raw := &lsv1alpha1.Target{}
		targetName = lsv1alpha1helper.GenerateDataObjectName(contextName, targetName)
		if err := kubeClient.Get(ctx, kubernetes.ObjectKey(targetName, inst.Namespace), raw); err != nil {
			return nil, err
		}
		targets[i] = *raw
	}
	targetList, err := dataobjects.NewFromTargetList(targets)
	if err != nil {
		return nil, err
	}
	targetList.Def = &targetImport

	return targetList, nil
}

// GetTargetListImportBySelector fetches the target imports from the cluster, based on a label selector.
// If restrictToImport is true, a label selector will be added which fetches only targets that are marked as import.
func GetTargetListImportBySelector(
	ctx context.Context,
	kubeClient client.Client,
	contextName string,
	inst *lsv1alpha1.Installation,
	selector map[string]string,
	targetImport lsv1alpha1.TargetImport,
	restrictToImport bool) (*dataobjects.TargetList, error) {
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
	if err := kubeClient.List(ctx, targets, client.InNamespace(inst.Namespace), &client.ListOptions{LabelSelector: contextSelector}); err != nil {
		return nil, err
	}
	targetList, err := dataobjects.NewFromTargetList(targets.Items)
	if err != nil {
		return nil, err
	}
	targetList.Def = &targetImport
	return targetList, nil
}

// GetComponentDescriptorImport fetches the component descriptor import from the cluster/registry.
func GetComponentDescriptorImport(ctx context.Context, kubeClient client.Client, contextName string, op *Operation, imp lsv1alpha1.ComponentDescriptorImport) (*dataobjects.ComponentDescriptor, error) {
	var refType dataobjects.CDReferenceType
	if imp.ConfigMapRef != nil {
		refType = dataobjects.ConfigMapReference
	} else if imp.SecretRef != nil {
		refType = dataobjects.SecretReference
	} else if imp.Ref != nil {
		refType = dataobjects.RegistryReference
	} else if len(imp.DataRef) != 0 {
		refType = dataobjects.DataReference
	} else {
		return nil, fmt.Errorf("invalid component descriptor import '%s': none of dataRef, configMapRef, secretRef, and componentDescriptorRef is specified", imp.Name)
	}

	res := dataobjects.NewComponentDescriptor()
	res.Def = &imp
	owner := kubernetes.GetOwner(op.Inst.Info.ObjectMeta)
	if OwnerReferenceIsInstallation(owner) {
		res.SetOwner(owner)
	}
	switch refType {
	case dataobjects.DataReference:
		// resolving data references is hard at this point, therefore they are replaced during the subinstallation template rendering
		// this means that there shouldn't be any data reference at this point
		return nil, fmt.Errorf("unsupported reference type '%s'", string(refType))
	case dataobjects.RegistryReference:
		// fetch component descriptor from registry
		if imp.Ref == nil {
			return nil, fmt.Errorf("reference type mismatch: reference type is '%s', but Ref is nil", string(refType))
		}
		cd, err := op.ComponentsRegistry().Resolve(ctx, imp.Ref.RepositoryContext, imp.Ref.ComponentName, imp.Ref.Version)
		if err != nil {
			return nil, fmt.Errorf("unable to get component descriptor from registry (%v): %w", imp.Ref, err)
		}
		res.SetRegistryReference(imp.Ref).SetDescriptor(cd)
	case dataobjects.ConfigMapReference:
		_, yamlData, _, err := ResolveConfigMapReference(ctx, kubeClient, imp.ConfigMapRef)
		if err != nil {
			return nil, fmt.Errorf("unable to get component descriptor from configmap %s: %w", imp.ConfigMapRef.NamespacedName().String(), err)
		}
		data, err := yaml.ToJSON([]byte(yamlData))
		if err != nil {
			return nil, fmt.Errorf("unable to convert yaml data to json: %w", err)
		}
		cd := &cdv2.ComponentDescriptor{}
		err = json.Unmarshal([]byte(data), cd)
		if err != nil {
			return nil, fmt.Errorf("unable to convert data into component descriptor: %w", err)
		}
		res.SetConfigMapReference(imp.ConfigMapRef).SetDescriptor(cd)
	case dataobjects.SecretReference:
		_, yamlData, _, err := ResolveSecretReference(ctx, kubeClient, imp.SecretRef)
		if err != nil {
			return nil, fmt.Errorf("unable to get component descriptor from secret %s: %w", imp.SecretRef.NamespacedName().String(), err)
		}
		data, err := yaml.ToJSON(yamlData)
		if err != nil {
			return nil, fmt.Errorf("unable to convert yaml data to json: %w", err)
		}
		cd := &cdv2.ComponentDescriptor{}
		err = json.Unmarshal(data, cd)
		if err != nil {
			return nil, fmt.Errorf("unable to convert data into component descriptor: %w", err)
		}
		res.SetSecretReference(imp.SecretRef).SetDescriptor(cd)
	default:
		return nil, fmt.Errorf("unknown reference type '%s' for component descriptor import", string(refType))
	}

	return res, nil
}

// GetComponentDescriptorListImport fetches all component descriptor imports in the list from the cluster/registry.
func GetComponentDescriptorListImport(ctx context.Context, kubeClient client.Client, contextName string, op *Operation, imp lsv1alpha1.ComponentDescriptorImport) (*dataobjects.ComponentDescriptorList, error) {
	// verify that the import describes a component descriptor list
	if imp.List == nil {
		return nil, fmt.Errorf("invalid component descriptor list import %s: import does not describe a list", imp.Name)
	}
	res := dataobjects.NewComponentDescriptorListWithSize(len(imp.List))
	res.Def = &imp
	for i, elem := range imp.List {
		dummyImportDef := lsv1alpha1.ComponentDescriptorImport{
			Name:         fmt.Sprintf("%s[%d]", imp.Name, i),
			Ref:          elem.Ref,
			ConfigMapRef: elem.ConfigMapRef,
			SecretRef:    elem.SecretRef,
			DataRef:      elem.DataRef,
		}
		cd, err := GetComponentDescriptorImport(ctx, kubeClient, contextName, op, dummyImportDef)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve component descriptor for index %d of cd import list %s: %w", i, imp.Name, err)
		}
		res.ComponentDescriptors[i] = cd
	}
	return res, nil
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
func ByteMapToRawMessageMap(m map[string][]byte) (map[string]json.RawMessage, error) {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		jsonVal, err := yaml.ToJSON(val)
		if err != nil {
			return nil, err
		}
		n[key] = json.RawMessage(jsonVal)
	}
	return n, nil
}

// StringMapToRawMessageMap converts a map of strings to a map of json.RawMessages
func StringMapToRawMessageMap(m map[string]string) (map[string]json.RawMessage, error) {
	n := make(map[string]json.RawMessage, len(m))
	for key, val := range m {
		jsonVal, err := yaml.ToJSON([]byte(val))
		if err != nil {
			return nil, err
		}
		n[key] = json.RawMessage(jsonVal)
	}
	return n, nil
}

// ResolveSecretReference is an auxiliary function that fetches the content of a secret as specified by the given SecretReference
// The first returned value is the complete secret content, the second one the specified key (if set), the third one is the generation of the secret
func ResolveSecretReference(ctx context.Context, kubeClient client.Client, secretRef *lsv1alpha1.SecretReference) (map[string][]byte, []byte, int64, error) {
	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, secretRef.NamespacedName(), secret); err != nil {
		return nil, nil, 0, err
	}
	completeData := secret.Data
	var (
		data   []byte
		ok     bool
		rawMap map[string]json.RawMessage
		err    error
	)
	if len(secretRef.Key) != 0 {
		data, ok = secret.Data[secretRef.Key]
		if !ok {
			return nil, nil, 0, fmt.Errorf("key %s in secret %s does not exist", secretRef.Key, secretRef.NamespacedName().String())
		}
	} else {
		// use the whole secret as map
		rawMap, err = ByteMapToRawMessageMap(secret.Data)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("unable to convert secret data to raw message map: %w", err)
		}
		data, err = json.Marshal(rawMap)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("unable to marshal secret data as map: %w", err)
		}
	}

	return completeData, data, secret.Generation, nil
}

// ResolveConfigMapReference is an auxiliary function that fetches the content of a configmap as specified by the given ConfigMapReference
// The first returned value is the complete configmap content, the second one the specified key (if set), the third one is the generation of the configmap
func ResolveConfigMapReference(ctx context.Context, kubeClient client.Client, configMapRef *lsv1alpha1.ConfigMapReference) (map[string][]byte, []byte, int64, error) {
	cm := &corev1.ConfigMap{}
	if err := kubeClient.Get(ctx, configMapRef.NamespacedName(), cm); err != nil {
		return nil, nil, 0, err
	}
	completeData := cm.BinaryData
	if completeData == nil {
		completeData = map[string][]byte{}
	}
	for k, v := range cm.Data {
		// kubernetes verifies that this doesn't cause any collisions
		completeData[k] = []byte(v)
	}
	var (
		data   []byte
		sdata  string
		rawMap map[string]json.RawMessage
		err    error
	)
	keyFound := len(configMapRef.Key) == 0
	if cm.Data != nil {
		if len(configMapRef.Key) != 0 {
			sdata, keyFound = cm.Data[configMapRef.Key]
			data = []byte(sdata)
		} else {
			// use whole configmap as json object
			rawMap, err := StringMapToRawMessageMap(cm.Data)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to convert configmap data to raw message map: %w", err)
			}
			data, err = json.Marshal(rawMap)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to marshal configmap data as map: %w", err)
			}
		}
	}
	if cm.BinaryData != nil {
		if len(configMapRef.Key) != 0 {
			data, keyFound = cm.BinaryData[configMapRef.Key]
		} else {
			// use whole configmap as json object
			rawMap, err = ByteMapToRawMessageMap(cm.BinaryData)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to convert configmap data to raw message map: %w", err)
			}
			data, err = json.Marshal(rawMap)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("unable to marshal configmap data as map: %w", err)
			}
		}
	}
	if !keyFound {
		return nil, nil, 0, fmt.Errorf("key '%s' in configmap '%s' does not exist", configMapRef.Key, configMapRef.NamespacedName().String())
	}

	return completeData, data, cm.Generation, nil
}

func OwnerReferenceIsInstallation(owner *metav1.OwnerReference) bool {
	return owner != nil && owner.Kind == "Installation"
}

func OwnerReferenceIsInstallationButNoParent(owner *metav1.OwnerReference, installation *lsv1alpha1.Installation) bool {
	parentName := GetParentInstallationName(installation)

	return OwnerReferenceIsInstallation(owner) && owner.Name != parentName

}
