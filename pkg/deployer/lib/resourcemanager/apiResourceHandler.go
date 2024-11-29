package resourcemanager

import (
	"fmt"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"github.com/gardener/landscaper/apis/core/v1alpha1"
	lserror "github.com/gardener/landscaper/apis/errors"
)

type ApiResourceHandler struct {
	clientset kubernetes.Interface

	// internal cache for api resources where the key is the GroupVersionKind string
	apiResourcesCache map[string]metav1.APIResource

	rwLock sync.RWMutex
}

func CreateApiResourceHandler(clientset kubernetes.Interface) *ApiResourceHandler {
	return &ApiResourceHandler{
		clientset:         clientset,
		apiResourcesCache: make(map[string]metav1.APIResource),
	}
}

func (a *ApiResourceHandler) getSyncFromCache(gkv string) (metav1.APIResource, bool) {
	a.rwLock.RLock()
	defer a.rwLock.RUnlock()
	res, ok := a.apiResourcesCache[gkv]
	return res, ok
}

func (a *ApiResourceHandler) GetApiResource(manifest *Manifest) (metav1.APIResource, error) {
	return a.GetApiResourceForType(manifest.TypeMeta)
}

func (a *ApiResourceHandler) GetApiResourceForType(typeMeta metav1.TypeMeta) (metav1.APIResource, error) {
	currOp := "GetApiResource"
	gvk := typeMeta.GetObjectKind().GroupVersionKind().String()

	// check if in cache
	if res, ok := a.getSyncFromCache(gvk); ok {
		return res, nil
	}

	a.rwLock.Lock()
	defer a.rwLock.Unlock()

	// recheck if now in cache
	if res, ok := a.apiResourcesCache[gvk]; ok {
		return res, nil
	}

	groupVersion := typeMeta.GetObjectKind().GroupVersionKind().GroupVersion().String()
	kind := typeMeta.GetObjectKind().GroupVersionKind().Kind
	apiResourceList, err := a.clientset.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		err2 := fmt.Errorf("unable to get api resources for %s: %w", groupVersion, err)
		if apierrors.IsUnauthorized(err) {
			// In case of an unauthorized error, the DeployItem should fail directly. This is achieved by the
			// error code v1alpha1.ErrorUnauthorized, which belongs to the v1alpha1.UnrecoverableErrorCodes.
			return metav1.APIResource{}, lserror.NewWrappedError(err2, currOp, "GetApiResourceList", err2.Error(), v1alpha1.ErrorUnauthorized)
		}
		return metav1.APIResource{}, lserror.NewWrappedError(err2, currOp, "GetApiResourceList", err2.Error())
	}

	for _, apiResource := range apiResourceList.APIResources {
		groupVersionKind := schema.GroupVersionKind{
			Group:   apiResource.Group,
			Version: apiResource.Version,
			Kind:    apiResource.Kind,
		}

		a.apiResourcesCache[groupVersionKind.String()] = apiResource
		if apiResource.Kind == kind {
			return apiResource, nil
		}
	}

	err = fmt.Errorf("unable to get apiresource for %s", gvk)
	return metav1.APIResource{}, lserror.NewWrappedError(err, currOp, "GetApiResource", err.Error())
}
