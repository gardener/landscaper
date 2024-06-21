package types

import lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

type ComponentVersionKey struct {
	Name    string
	Version string
}

func (k *ComponentVersionKey) String() string {
	if k == nil {
		return ""
	}
	return k.Name + ":" + k.Version
}

func ComponentVersionKeyFromReference(cdRef *lsv1alpha1.ComponentDescriptorReference) *ComponentVersionKey {
	if cdRef == nil {
		return nil
	}
	return &ComponentVersionKey{
		Name:    cdRef.ComponentName,
		Version: cdRef.Version,
	}
}
