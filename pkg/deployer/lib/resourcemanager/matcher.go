package resourcemanager

import (
	"fmt"
	"slices"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
)

type Matcher interface {
	Match(*managedresource.ManagedResourceStatus) bool
}

func newPredefinedMatcher(predefined *managedresource.PredefinedResourceGroup) (Matcher, error) {
	switch predefined.Type {
	case managedresource.PredefinedResourceGroupNamespacedResources:
		return &NamespacedMatcher{}, nil
	case managedresource.PredefinedResourceGroupClusterScopedResources:
		return &ClusterScopedMatcher{}, nil
	case managedresource.PredefinedResourceGroupCRDs:
		return &CRDMatcher{}, nil
	case managedresource.PredefinedResourceGroupEmpty:
		return &EmptyMatcher{}, nil
	default:
		return nil, fmt.Errorf("invalid deletion group: unsupported type of predefinedResourceGroup: %s", predefined.Type)
	}
}

func isCRD(res *managedresource.ManagedResourceStatus) bool {
	return res.Resource.Kind == "CustomResourceDefinition"
}

type NamespacedMatcher struct{}

func (m *NamespacedMatcher) Match(res *managedresource.ManagedResourceStatus) bool {
	return !isCRD(res) && len(res.Resource.Namespace) > 0
}

type ClusterScopedMatcher struct{}

func (m *ClusterScopedMatcher) Match(res *managedresource.ManagedResourceStatus) bool {
	return !isCRD(res) && len(res.Resource.Namespace) == 0
}

type CRDMatcher struct{}

func (m *CRDMatcher) Match(res *managedresource.ManagedResourceStatus) bool {
	return isCRD(res)
}

type EmptyMatcher struct{}

func (m *EmptyMatcher) Match(*managedresource.ManagedResourceStatus) bool {
	return false
}

func newCustomMatcher(custom *managedresource.CustomResourceGroup) Matcher {
	return &CustomMatcher{
		resourceTypes: custom.Resources,
	}
}

type CustomMatcher struct {
	resourceTypes []managedresource.ResourceType
}

func (m *CustomMatcher) Match(res *managedresource.ManagedResourceStatus) bool {
	for _, t := range m.resourceTypes {
		if t.Kind == res.Resource.Kind &&
			t.APIVersion == res.Resource.APIVersion &&
			listIsEmptyOrContainsElement(t.Namespaces, res.Resource.Namespace) &&
			listIsEmptyOrContainsElement(t.Names, res.Resource.Name) {

			return true
		}
	}

	return false
}

func listIsEmptyOrContainsElement(list []string, element string) bool {
	return len(list) == 0 || slices.Contains(list, element)
}
