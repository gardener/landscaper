package lib

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
)

func ExpandManagedResourceManifests(origManifests []managedresource.Manifest) ([]managedresource.Manifest, error) {
	result := []managedresource.Manifest{}

	for i := range origManifests {
		origManifest := &origManifests[i]
		rawExtensions, err := expandManifest(origManifest.Manifest)
		if err != nil {
			return nil, fmt.Errorf("unable to expand managed resource manifests: %w", err)
		}

		for k := range rawExtensions {
			result = append(result, managedresource.Manifest{
				Policy:               origManifest.Policy,
				Manifest:             rawExtensions[k],
				AnnotateBeforeCreate: origManifest.AnnotateBeforeCreate,
				AnnotateBeforeDelete: origManifest.AnnotateBeforeDelete,
			})
		}
	}

	return result, nil
}

func ExpandManifests(manifests []*runtime.RawExtension) ([]*runtime.RawExtension, error) {
	result := []*runtime.RawExtension{}

	for i := range manifests {
		manifestList, err := expandManifest(manifests[i])
		if err != nil {
			return nil, err
		}

		result = append(result, manifestList...)
	}

	return result, nil
}

func expandManifest(manifest *runtime.RawExtension) ([]*runtime.RawExtension, error) {
	manifestList := &metav1.List{}
	if err := json.Unmarshal(manifest.Raw, &manifestList); err != nil {
		return nil, fmt.Errorf("unable to expand manifest: %w", err)
	}

	if len(manifestList.Items) == 0 {
		return []*runtime.RawExtension{manifest}, nil
	}

	result := make([]*runtime.RawExtension, len(manifestList.Items))
	for i := range manifestList.Items {
		result[i] = &manifestList.Items[i]
	}
	return result, nil
}
