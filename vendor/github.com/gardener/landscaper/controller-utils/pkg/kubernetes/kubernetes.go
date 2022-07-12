// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// image err reasons
// defined in https://github.com/kubernetes/kubernetes/blob/cea1d4e20b4a7886d8ff65f34c6d4f95efcb4742/pkg/kubelet/images/types.go

// ErrImagePull - General image pull error
const ErrImagePull = "ErrImagePull"

// ErrImagePullBackOff - Container image pull failed, kubelet is backing off image pull
const ErrImagePullBackOff = "ImagePullBackOff"

// ErrImageNeverPull - Required Image is absent on host and PullPolicy is NeverPullImage
const ErrImageNeverPull = "ErrImageNeverPull"

// ErrRegistryUnavailable - Get http error when pulling image from registry
const ErrRegistryUnavailable = "RegistryUnavailable"

// ErrInvalidImageName - Unable to parse the image name.
const ErrInvalidImageName = "ErrInvalidImageName"

// CreateOrUpdate creates or updates the given object in the Kubernetes
// cluster. The object's desired state must be reconciled with the existing
// state inside the passed in callback MutateFn.
// It also correctly handles objects that have the generateName attribute set.
//
// The MutateFn is called regardless of creating or updating an object.
//
// It returns the executed operation and an error.
func CreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	// check if the name key has to be generated
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	key := client.ObjectKey{Namespace: accessor.GetNamespace(), Name: accessor.GetName()}

	if accessor.GetName() == "" && accessor.GetGenerateName() != "" {
		if err := Mutate(f, key, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := c.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	return controllerutil.CreateOrUpdate(ctx, c, obj, f)
}

// AddServiceAccountAuth adds the authentication information of a service account to a rest config.
// It also waits for the secret with the token to be created.
func AddServiceAccountAuth(ctx context.Context, kubeClient client.Client, sa *corev1.ServiceAccount, config *rest.Config) error {
	saKey := ObjectKeyFromObject(sa)
	// wait for k8s to generate the service account secret
	err := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (done bool, err error) {
		if err := kubeClient.Get(ctx, saKey, sa); err != nil {
			return false, nil
		}
		return len(sa.Secrets) != 0, nil
	})
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	if err := kubeClient.Get(ctx, ObjectKey(sa.Secrets[0].Name, sa.Namespace), secret); err != nil {
		return fmt.Errorf("unable to get service account %q secret: %w", sa.Name, err)
	}

	if token, ok := secret.Data[corev1.ServiceAccountTokenKey]; ok {
		config.BearerToken = string(token)
		return nil

	}
	return fmt.Errorf("no token for authentication was present in the secret %q", secret.Name)
}

// AddServiceAccountToken adds the authentication information of a service account to a rest config.
// It also waits until the token has been created.
func AddServiceAccountToken(ctx context.Context, kubeClient client.Client, sa *corev1.ServiceAccount, config *rest.Config) error {
	secrets, err := GetSecretsForServiceAccount(ctx, kubeClient, ObjectKeyFromObject(sa))
	if err != nil {
		return fmt.Errorf("unable to get secrets for service account %s: %w", sa.Name, err)
	}

	var secret *corev1.Secret
	if len(secrets) == 0 {
		secret, err = CreateSecretForServiceAccount(ctx, kubeClient, sa)
		if err != nil {
			return fmt.Errorf("unable to create secret for service account %s: %w", sa.Name, err)
		}
	} else {
		secret = secrets[0]
	}

	if err = WaitForServiceAccountToken(ctx, kubeClient, ObjectKeyFromObject(secret)); err != nil {
		return fmt.Errorf("timeout when waiting for token of service account %s: %w", sa.Name, err)
	}

	if err = kubeClient.Get(ctx, ObjectKeyFromObject(secret), secret); err != nil {
		return fmt.Errorf("unable to read secret %s for service account %s: %w", secret.Name, sa.Name, err)
	}

	token, ok := secret.Data[corev1.ServiceAccountTokenKey]
	if !ok {
		return fmt.Errorf("no token for authentication was present in the secret %q", secret.Name)
	}

	config.BearerToken = string(token)
	return nil
}

// ResolveSecrets finds and returns the secrets referenced by secretRefs
func ResolveSecrets(ctx context.Context, client client.Client, secretRefs []v1alpha1.ObjectReference) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, len(secretRefs))
	for i, secretRef := range secretRefs {
		secret := corev1.Secret{}
		// todo: check for cache
		if err := client.Get(ctx, secretRef.NamespacedName(), &secret); err != nil {
			return nil, err
		}
		secrets[i] = secret
	}
	return secrets, nil
}

// Mutate wraps a MutateFn and applies validation to its result
func Mutate(f controllerutil.MutateFn, key client.ObjectKey, obj client.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey := client.ObjectKeyFromObject(obj); key != newKey {
		return fmt.Errorf("MutateFn cannot Mutate object name and/or object namespace")
	}
	return nil
}

// ObjectKey creates a namespaced name (client.ObjectKey) for a given name and namespace.
func ObjectKey(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

// ObjectKeyFromObject creates a namespaced name for a given object.
func ObjectKeyFromObject(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// ReconcileRequestFromObject creates a namespaced name for a given object.
func ReconcileRequestFromObject(obj client.Object) reconcile.Request {
	return reconcile.Request{
		NamespacedName: ObjectKeyFromObject(obj),
	}
}

// GetStatusForContainer returns the container status for a specific container
func GetStatusForContainer(containerStatus []corev1.ContainerStatus, name string) (corev1.ContainerStatus, error) {
	for _, status := range containerStatus {
		if status.Name == name {
			return status, nil
		}
	}
	return corev1.ContainerStatus{}, errors.New("container not found")
}

// OwnerOfGVK validates whether a instance of the given gvk is referenced
func OwnerOfGVK(ownerRefs []metav1.OwnerReference, gvk schema.GroupVersionKind) (string, bool) {
	for _, ownerRef := range ownerRefs {
		gv, err := schema.ParseGroupVersion(ownerRef.APIVersion)
		if err != nil {
			continue
		}
		if gv.Group == gvk.Group && ownerRef.Kind == gvk.Kind {
			return ownerRef.Name, true
		}
	}
	return "", false
}

// GetOwner returns the owner reference of a object.
// If multiple owners are defined, the controlling owner is returned.
func GetOwner(objMeta metav1.ObjectMeta) *metav1.OwnerReference {
	return GetMainOwnerFromOwnerReferences(objMeta.GetOwnerReferences())
}

// GetMainOwnerFromOwnerReferences returns the main owner from a list of owner references.
// If the list contains only a single reference, that one is returned.
// If multiple owners are defined, the controlling owner is returned.
// If there are multiple references and no controlling one, nil is returned.
func GetMainOwnerFromOwnerReferences(owners []metav1.OwnerReference) *metav1.OwnerReference {
	if len(owners) == 1 {
		return &owners[0]
	}
	for _, ownerRef := range owners {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			return &ownerRef
		}
	}
	return nil
}

// TypedObjectReferenceFromObject creates a typed object reference from a object.
func TypedObjectReferenceFromObject(obj runtime.Object, scheme *runtime.Scheme) (*v1alpha1.TypedObjectReference, error) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.TypedObjectReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		ObjectReference: v1alpha1.ObjectReference{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
		},
	}, nil
}

// TypedObjectReferenceFromUnstructuredObject creates a typed object reference from an unstructured object.
func TypedObjectReferenceFromUnstructuredObject(obj *unstructured.Unstructured) *v1alpha1.TypedObjectReference {
	return &v1alpha1.TypedObjectReference{
		APIVersion: obj.GroupVersionKind().GroupVersion().String(),
		Kind:       obj.GetKind(),
		ObjectReference: v1alpha1.ObjectReference{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

// CoreObjectReferenceFromUnstructuredObject creates a core object reference from an unstructured object.
func CoreObjectReferenceFromUnstructuredObject(obj *unstructured.Unstructured) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		APIVersion: obj.GroupVersionKind().GroupVersion().String(),
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		UID:        obj.GetUID(),
	}
}

// ObjectFromTypedObjectReference creates an unstructured object from a typed object reference.
func ObjectFromTypedObjectReference(ref *v1alpha1.TypedObjectReference) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": ref.APIVersion,
			"kind":       ref.Kind,
			"metadata": map[string]interface{}{
				"name":      ref.Name,
				"namespace": ref.Namespace,
			},
		},
	}
}

// ObjectFromCoreObjectReference creates an unstructured object from a core object reference.
func ObjectFromCoreObjectReference(ref *corev1.ObjectReference) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": ref.APIVersion,
			"kind":       ref.Kind,
			"metadata": map[string]interface{}{
				"name":      ref.Name,
				"namespace": ref.Namespace,
			},
		},
	}
}

// HasFinalizer checks if the object constains a finalizer with the given name.
func HasFinalizer(obj metav1.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}

// SetMetaDataLabel sets the label and value
func SetMetaDataLabel(obj metav1.Object, lab string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[lab] = value
	obj.SetLabels(labels)
}

// HasLabel checks if the objects has a label
func HasLabel(obj metav1.Object, lab string) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	_, ok := labels[lab]
	return ok
}

// HasLabelWithValue checks if the objects has a label with a value
func HasLabelWithValue(obj metav1.Object, lab string, value string) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	val, ok := labels[lab]
	if !ok {
		return false
	}
	return val == value
}

// ConvertToRawExtension converts a object to a raw extension.
// The type of the object is automatically set given the scheme.
func ConvertToRawExtension(from runtime.Object, scheme *runtime.Scheme) (*runtime.RawExtension, error) {
	// set the apiversion and kind
	if err := InjectTypeInformation(from, scheme); err != nil {
		return nil, err
	}

	fromBytes, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	ext := &runtime.RawExtension{
		Raw:    fromBytes,
		Object: from,
	}
	return ext, nil
}

// InjectTypeInformation injects the group version kind into a runtime object.
// This is needed as this information gets lost when a struct is decoded.
func InjectTypeInformation(obj runtime.Object, scheme *runtime.Scheme) error {
	// set the apiversion and kind
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return fmt.Errorf("unable to to get gvk for provider configuration: %w", err)
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	return nil
}

// GenerateKubeconfigBytes generates a kubernetes kubeconfig config object from a rest config
// and encodes it as yaml.
func GenerateKubeconfigBytes(restConfig *rest.Config) ([]byte, error) {
	clientConfig, err := GenerateKubeconfig(restConfig)
	if err != nil {
		return nil, err
	}
	return clientcmd.Write(clientConfig)
}

// GenerateKubeconfigJSONBytes generates a kubernetes kubeconfig config object from a rest config
// and encodes it as json.
func GenerateKubeconfigJSONBytes(restConfig *rest.Config) ([]byte, error) {
	clientConfig, err := GenerateKubeconfig(restConfig)
	if err != nil {
		return nil, err
	}
	data, err := clientcmd.Write(clientConfig)
	if err != nil {
		return nil, err
	}
	return yaml.YAMLToJSON(data)
}

// GenerateKubeconfig generates a kubernetes kubeconfig config object from a rest config
func GenerateKubeconfig(restConfig *rest.Config) (clientcmdapi.Config, error) {
	const defaultID = "default"

	var (
		caData   = restConfig.TLSClientConfig.CAData
		certData = restConfig.TLSClientConfig.CertData
		keyData  = restConfig.TLSClientConfig.KeyData

		bearerToken = restConfig.BearerToken
	)
	if len(restConfig.TLSClientConfig.CAFile) != 0 {
		data, err := ioutil.ReadFile(restConfig.TLSClientConfig.CAFile)
		if err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("unable to read ca from %q: %w", restConfig.TLSClientConfig.CAFile, err)
		}
		caData = data
	}
	if len(restConfig.TLSClientConfig.CertFile) != 0 {
		data, err := ioutil.ReadFile(restConfig.TLSClientConfig.CertFile)
		if err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("unable to read cert from %q: %w", restConfig.TLSClientConfig.CertFile, err)
		}
		certData = data
	}
	if len(restConfig.TLSClientConfig.KeyFile) != 0 {
		data, err := ioutil.ReadFile(restConfig.TLSClientConfig.KeyFile)
		if err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("unable to read key from %q: %w", restConfig.TLSClientConfig.KeyFile, err)
		}
		keyData = data
	}
	if len(restConfig.BearerTokenFile) != 0 {
		data, err := ioutil.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("unable to read bearer token from %q: %w", restConfig.BearerTokenFile, err)
		}
		bearerToken = string(data)
	}

	cfg := clientcmdapi.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: defaultID,
		Contexts: map[string]*clientcmdapi.Context{
			defaultID: {
				Cluster:  defaultID,
				AuthInfo: defaultID,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			defaultID: {
				Token: bearerToken,

				Username: restConfig.Username,
				Password: restConfig.Password,

				ClientCertificateData: certData,
				ClientKeyData:         keyData,
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			defaultID: {
				Server:                   restConfig.Host + restConfig.APIPath,
				CertificateAuthorityData: caData,
				InsecureSkipTLSVerify:    restConfig.Insecure,
			},
		},
	}
	return cfg, nil
}

// ParseFiles parses a map of filename->data into unstructured yaml objects.
func ParseFiles(log logr.Logger, files map[string]string) ([]*unstructured.Unstructured, error) {
	objects := make([]*unstructured.Unstructured, 0)
	for name, content := range files {
		if _, file := filepath.Split(name); file == "NOTES.txt" {
			continue
		}
		decodedObjects, err := DecodeObjects(log, name, []byte(content))
		if err != nil {
			return nil, fmt.Errorf("unable to decode files for %q: %w", name, err)
		}
		objects = append(objects, decodedObjects...)
	}
	return objects, nil
}

// ParseFilesToRawExtension parses a map of filename->data into unstructured yaml objects.
func ParseFilesToRawExtension(log logr.Logger, files map[string]string) ([]*runtime.RawExtension, error) {
	objects := make([]*runtime.RawExtension, 0)
	for name, content := range files {
		if _, file := filepath.Split(name); file == "NOTES.txt" {
			continue
		}
		decodedObjects, err := DecodeObjectsToRawExtension(log, name, []byte(content))
		if err != nil {
			return nil, fmt.Errorf("unable to decode files for %q: %w", name, err)
		}
		objects = append(objects, decodedObjects...)
	}
	return objects, nil
}

// DecodeObjects decodes raw data that can be a multiyaml file into unstructured kubernetes objects.
func DecodeObjects(log logr.Logger, name string, data []byte) ([]*unstructured.Unstructured, error) {
	var (
		decoder    = yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024)
		decodedObj map[string]interface{}
		objects    = make([]*unstructured.Unstructured, 0)
	)

	for i := 0; true; i++ {
		if err := decoder.Decode(&decodedObj); err != nil {
			if err == io.EOF {
				break
			}
			log.Error(err, fmt.Sprintf("unable to decode resource %d of file %q", i, name))
			continue
		}
		if decodedObj == nil {
			continue
		}
		obj := &unstructured.Unstructured{Object: decodedObj}
		// ignore the obj if no group version is defined
		if len(obj.GetAPIVersion()) == 0 {
			continue
		}
		objects = append(objects, obj.DeepCopy())
	}
	return objects, nil
}

// DecodeObjectsToRawExtension decodes raw data that can be a multiyaml file into unstructured kubernetes objects.
func DecodeObjectsToRawExtension(log logr.Logger, name string, data []byte) ([]*runtime.RawExtension, error) {
	var (
		decoder = yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024)
		objects = make([]*runtime.RawExtension, 0)
	)

	for i := 0; true; i++ {
		obj := &runtime.RawExtension{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			log.Error(err, fmt.Sprintf("unable to decode resource %d of file %q", i, name))
			continue
		}
		if obj == nil || obj.Raw == nil {
			continue
		}
		// ignore the obj if no group version is defined
		var typeMeta runtime.TypeMeta
		if err := json.Unmarshal(obj.Raw, &typeMeta); err != nil {
			log.Error(err, fmt.Sprintf("unable to parse type meta %d of file %q to runtime object", i, name))
			continue
		}
		if typeMeta.GetObjectKind().GroupVersionKind().Empty() {
			continue
		}
		objects = append(objects, obj.DeepCopy())
	}
	return objects, nil
}

// DeleteAndWaitForObjectDeleted deletes an object and waits for the object to be deleted.
func DeleteAndWaitForObjectDeleted(ctx context.Context, kubeClient client.Client, timeout time.Duration, obj client.Object) error {
	if err := kubeClient.Delete(ctx, obj); err != nil {
		gvk := obj.GetObjectKind().GroupVersionKind().String()
		return fmt.Errorf("unable to delete %s %s/%s: %w", gvk, obj.GetName(), obj.GetNamespace(), err)
	}

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	delCondFunc := GenerateDeleteObjectConditionFunc(ctx, kubeClient, obj)
	return wait.PollImmediateUntil(5*time.Second, delCondFunc, pollCtx.Done())
}

// GenerateDeleteObjectConditionFunc creates a condition function to validate the deletion of objects.
func GenerateDeleteObjectConditionFunc(ctx context.Context, kubeClient client.Client, obj client.Object) wait.ConditionFunc {
	return func() (done bool, err error) {
		key := ObjectKey(obj.GetName(), obj.GetNamespace())
		if err := kubeClient.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

// SetRequiredNestedFieldsFromObj sets the immutable or the fields that are required
// when updating un object. e.g: the ClusterIP of a service.
func SetRequiredNestedFieldsFromObj(currObj, obj *unstructured.Unstructured) error {
	currObjGK := currObj.GroupVersionKind().GroupKind()
	objGK := obj.GroupVersionKind().GroupKind()

	if currObjGK != objGK {
		return fmt.Errorf("objects do not have the same GoupKind: %s/%s", currObjGK, objGK)
	}

	switch currObjGK.String() {
	case "Job.batch":
		selector, found, err := unstructured.NestedMap(currObj.Object, "spec", "selector")
		if err != nil {
			return err
		}
		if found {
			if err := unstructured.SetNestedMap(obj.Object, selector, "spec", "selector"); err != nil {
				return err
			}
		}
		labels, found, err := unstructured.NestedMap(currObj.Object, "spec", "template", "metadata", "labels")
		if err != nil {
			return err
		}
		if found {
			if err := unstructured.SetNestedMap(obj.Object, labels, "spec", "template", "metadata", "labels"); err != nil {
				return err
			}
		}
	case "Service":
		clusterIP, found, err := unstructured.NestedString(currObj.Object, "spec", "clusterIP")
		if err != nil {
			return err
		}
		if found {
			if err := unstructured.SetNestedField(obj.Object, clusterIP, "spec", "clusterIP"); err != nil {
				return err
			}
		}
	case "ServiceAccount":
		secrets, found, err := unstructured.NestedSlice(obj.Object, "secrets")
		if err != nil {
			return err
		}

		if !found || len(secrets) == 0 {
			secrets, found, err = unstructured.NestedSlice(currObj.Object, "secrets")
			if err != nil {
				return err
			}
			if found && len(secrets) > 0 {
				if err := unstructured.SetNestedSlice(obj.Object, secrets, "secrets"); err != nil {
					return err
				}
			}
		}
	}

	resourceVersion, found, err := unstructured.NestedString(currObj.Object, "metadata", "resourceVersion")
	if err != nil {
		return err
	}
	if found {
		if err := unstructured.SetNestedField(obj.Object, resourceVersion, "metadata", "resourceVersion"); err != nil {
			return err
		}
	}

	return nil
}
