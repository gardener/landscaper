package resourcemanager

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
)

func DeleteManagedResources(
	ctx context.Context,
	managedResources managedresource.ManagedResourceStatusList,
	groupDefinitions []managedresource.DeletionGroupDefinition,
	targetClient client.Client,
	deployItem *lsv1alpha1.DeployItem,
	interruptionChecker interruption.InterruptionChecker,
) error {
	if len(managedResources) == 0 {
		return nil
	}

	// default group definitions
	if len(groupDefinitions) == 0 {
		groupDefinitions = defaultDeletionGroups()
	}

	// build groups
	groups := make([]*DeletionGroup, len(groupDefinitions))
	for i := range groupDefinitions {
		var err error
		groups[i], err = NewDeletionGroup(groupDefinitions[i], deployItem, targetClient, interruptionChecker)
		if err != nil {
			return err
		}
	}

	// divide resources into groups
	for i := range managedResources {
		res := &managedResources[i]
		for _, group := range groups {
			if group.Match(res) {
				group.AddResource(res)
				break
			}
		}
	}

	// delete groups
	for _, group := range groups {
		if err := group.Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

// defaultDeletionGroups defines the default order in which resources are deleted: first the namespaced resources,
// then the cluster-scoped resources (without CRDs), and finally the CRDs.
func defaultDeletionGroups() []managedresource.DeletionGroupDefinition {
	return []managedresource.DeletionGroupDefinition{
		{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
			Type: managedresource.PredefinedResourceGroupNamespacedResources,
		}},
		{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
			Type: managedresource.PredefinedResourceGroupClusterScopedResources,
		}},
		{PredefinedResourceGroup: &managedresource.PredefinedResourceGroup{
			Type: managedresource.PredefinedResourceGroupCRDs,
		}},
	}
}
