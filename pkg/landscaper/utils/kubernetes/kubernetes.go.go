package kubernetes

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
)

// OwnerOfGVK validates whether a instance of the given gvk is referenced
func OwnerOfGVK(ownerRefs []v1.OwnerReference, gvk schema.GroupVersionKind) (string, bool) {
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
func HasFinalizer(obj v1.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}
