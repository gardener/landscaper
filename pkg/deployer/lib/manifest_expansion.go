package lib

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
