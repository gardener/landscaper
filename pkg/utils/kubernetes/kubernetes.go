// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
)

// CreateOrUpdate creates or updates the given object in the Kubernetes
// cluster. The object's desired state must be reconciled with the existing
// state inside the passed in callback MutateFn.
// It also correctly handles objects that have the generateName attribute set.
//
// The MutateFn is called regardless of creating or updating an object.
//
// It returns the executed operation and an error.
func CreateOrUpdate(ctx context.Context, c client.Client, obj runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

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

// Mutate wraps a MutateFn and applies validation to its result
func Mutate(f controllerutil.MutateFn, key client.ObjectKey, obj runtime.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey, err := client.ObjectKeyFromObject(obj); err != nil || key != newKey {
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
	if len(objMeta.GetOwnerReferences()) == 1 {
		return &objMeta.GetOwnerReferences()[0]
	}
	for _, ownerRef := range objMeta.GetOwnerReferences() {
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

// GenerateKubeconfigBytes generates a kubernetes kubeconfig config object from a rest config
// and encodes it as yaml.
func GenerateKubeconfigBytes(restConfig *rest.Config) ([]byte, error) {
	return clientcmd.Write(GenerateKubeconfig(restConfig))
}

// GenerateKubeconfigJSONBytes generates a kubernetes kubeconfig config object from a rest config
// and encodes it as json.
func GenerateKubeconfigJSONBytes(restConfig *rest.Config) ([]byte, error) {
	data, err := clientcmd.Write(GenerateKubeconfig(restConfig))
	if err != nil {
		return nil, err
	}
	return yaml.YAMLToJSON(data)
}

// GenerateKubeconfig generates a kubernetes kubeconfig config object from a rest config
func GenerateKubeconfig(restConfig *rest.Config) clientcmdapi.Config {
	const defaultID = "default"
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
				Token: restConfig.BearerToken,

				Username: restConfig.Username,
				Password: restConfig.Password,

				ClientCertificateData: restConfig.CertData,
				ClientKeyData:         restConfig.KeyData,
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			defaultID: {
				Server:                   path.Join(restConfig.Host, restConfig.APIPath),
				CertificateAuthorityData: restConfig.CAData,
				InsecureSkipTLSVerify:    restConfig.Insecure,
			},
		},
	}
	return cfg
}

// ParseFiles parses a map of filename->data into unstructured yaml objects.
func ParseFiles(log logr.Logger, files map[string]string) ([]*unstructured.Unstructured, error) {
	objects := make([]*unstructured.Unstructured, 0)
	for name, content := range files {
		decodedObjects, err := DecodeObjects(log, name, []byte(content))
		if err != nil {
			return nil, fmt.Errorf("unable to decode files for %q: %w", name, err)
		}
		objects = append(objects, decodedObjects...)
	}
	return objects, nil
}

// Decodes raw data that can be a multiyaml file into unstructured kubernetes objects.
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
