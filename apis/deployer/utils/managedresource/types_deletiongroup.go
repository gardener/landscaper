package managedresource

type DeletionGroupDefinition struct {
	// +optional
	PredefinedResourceGroup *PredefinedResourceGroup `json:"predefinedResourceGroup,omitempty"`

	// +optional
	CustomResourceGroup *CustomResourceGroup `json:"customResourceGroup,omitempty"`
}

func (g *DeletionGroupDefinition) IsPredefined() bool {
	return g.PredefinedResourceGroup != nil
}

func (g *DeletionGroupDefinition) IsCustom() bool {
	return g.CustomResourceGroup != nil
}

type PredefinedResourceGroup struct {
	Type PredefinedResourceGroupType `json:"type,omitempty"`

	// +optional
	ForceDelete bool `json:"forceDelete,omitempty"`
}

type PredefinedResourceGroupType string

const (
	PredefinedResourceGroupNamespacedResources    PredefinedResourceGroupType = "namespaced-resources"
	PredefinedResourceGroupClusterScopedResources PredefinedResourceGroupType = "cluster-scoped-resources"
	PredefinedResourceGroupCRDs                   PredefinedResourceGroupType = "crds"
	PredefinedResourceGroupEmpty                  PredefinedResourceGroupType = "empty"
)

type CustomResourceGroup struct {
	Resources []ResourceType `json:"resources,omitempty"`

	// +optional
	ForceDelete bool `json:"forceDelete,omitempty"`

	// +optional
	DeleteAllResources bool `json:"deleteAllResources,omitempty"`
}

type ResourceType struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	// +optional
	Names []string `json:"names,omitempty"`
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
}
