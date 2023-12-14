package resourcemanager

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
)

const TimeoutCheckpointDeployerDeleteResources = "deployer: delete resources"

type DeletionGroup struct {
	definition          managedresource.DeletionGroupDefinition
	matcher             Matcher
	managedResources    []*managedresource.ManagedResourceStatus
	targetClient        client.Client
	deployItem          *lsv1alpha1.DeployItem
	interruptionChecker interruption.InterruptionChecker
}

func NewDeletionGroup(
	definition managedresource.DeletionGroupDefinition,
	deployItem *lsv1alpha1.DeployItem,
	targetClient client.Client,
	interruptionChecker interruption.InterruptionChecker,
) (group *DeletionGroup, err error) {
	if definition.IsPredefined() && definition.IsCustom() {
		return nil, fmt.Errorf("invalid deletion group: predefinedResourceGroup and customResourceGroup must not both be set")
	}
	if !definition.IsPredefined() && !definition.IsCustom() {
		return nil, fmt.Errorf("invalid deletion group: either predefinedResourceGroup or customResourceGroup must be set")
	}

	group = &DeletionGroup{
		definition:          definition,
		managedResources:    []*managedresource.ManagedResourceStatus{},
		deployItem:          deployItem,
		targetClient:        targetClient,
		interruptionChecker: interruptionChecker,
	}

	if definition.IsPredefined() {
		group.matcher, err = newPredefinedMatcher(definition.PredefinedResourceGroup)
		if err != nil {
			return nil, err
		}
	} else if definition.IsCustom() {
		group.matcher = newCustomMatcher(definition.CustomResourceGroup)
	}

	return group, nil
}

func (g *DeletionGroup) isForceDelete() bool {
	if g.definition.IsPredefined() {
		return g.definition.PredefinedResourceGroup.ForceDelete
	} else {
		return g.definition.CustomResourceGroup.ForceDelete
	}
}

func (g *DeletionGroup) isDeleteAllResources() bool {
	if g.definition.IsPredefined() {
		return false
	} else {
		return g.definition.CustomResourceGroup.DeleteAllResources
	}
}

// NORMAL MODE

func (g *DeletionGroup) Match(res *managedresource.ManagedResourceStatus) bool {
	return g.matcher.Match(res)
}

func (g *DeletionGroup) AddResource(res *managedresource.ManagedResourceStatus) {
	g.managedResources = append(g.managedResources, res)
}

func (g *DeletionGroup) GetManagedResources() []*managedresource.ManagedResourceStatus {
	return g.managedResources
}

// DELETE-ALL MODE

func (g *DeletionGroup) GetAllResources(ctx context.Context) ([]*managedresource.ManagedResourceStatus, error) {
	resources := []*managedresource.ManagedResourceStatus{}

	for _, t := range g.definition.CustomResourceGroup.Resources {
		resourcesForType, err := g.getAllResourcesForType(ctx, t)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resourcesForType...)
	}

	return resources, nil
}

func (g *DeletionGroup) getAllResourcesForType(ctx context.Context, t managedresource.ResourceType) ([]*managedresource.ManagedResourceStatus, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyResourceVersion, t.APIVersion,
		lc.KeyResourceKind, t.Kind)

	resources := []*managedresource.ManagedResourceStatus{}
	typeMeta := metav1.TypeMeta{APIVersion: t.APIVersion, Kind: t.Kind}

	isNamespaced, err := g.isNamespaced(typeMeta)
	if err != nil {
		log.Error(err, "deletiongroups: failed to determine whether type is namespaced")
		err = fmt.Errorf("deletiongroups: failed to determine whether type %v is namespaced: %w", typeMeta, err)
		return nil, err
	}

	if isNamespaced {
		// Handle either the specified namespaces, or all namespaces.
		// Per namespace, add all resources for the type and namespace.
		namespaces := t.Namespaces
		if len(t.Namespaces) == 0 {
			namespaces, err = g.listNamespaces(ctx)
			if err != nil {
				return nil, err
			}
		}

		for _, namespace := range namespaces {
			resourcesInNamespace, err := g.listResources(ctx, typeMeta, client.InNamespace(namespace))
			if err != nil {
				return nil, err
			}
			resourcesInNamespace = g.filterResourcesByNames(resourcesInNamespace, t.Names)
			resources = append(resources, resourcesInNamespace...)
		}

	} else if len(t.Names) > 0 {
		// Add cluster-scoped resources with the given names
		for _, name := range t.Names {
			resources = append(resources, &managedresource.ManagedResourceStatus{
				Resource: corev1.ObjectReference{
					APIVersion: t.APIVersion,
					Kind:       t.Kind,
					Name:       name,
				},
			})
		}

	} else {
		// Add all cluster-scoped resources of the given type.
		clusterScopedResources, err := g.listResources(ctx, typeMeta)
		if err != nil {
			return nil, err
		}
		resources = append(resources, clusterScopedResources...)
	}

	return resources, nil
}

func (g *DeletionGroup) isNamespaced(typeMeta metav1.TypeMeta) (bool, error) {
	u := &unstructured.Unstructured{}
	u.SetKind(typeMeta.Kind)
	u.SetAPIVersion(typeMeta.APIVersion)
	return g.targetClient.IsObjectNamespaced(u)
}

func (g *DeletionGroup) listNamespaces(ctx context.Context) ([]string, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	metadataList := &metav1.PartialObjectMetadataList{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}}
	if err := g.targetClient.List(ctx, metadataList); err != nil {
		log.Error(err, "deletiongroups: error listing namespaces")
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}

	namespaceList := convertList(metadataList.Items, func(item *metav1.PartialObjectMetadata) string {
		return item.Name
	})

	return namespaceList, nil
}

func (g *DeletionGroup) listResources(ctx context.Context, typeMeta metav1.TypeMeta, opts ...client.ListOption) ([]*managedresource.ManagedResourceStatus, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	metadataList := &metav1.PartialObjectMetadataList{TypeMeta: typeMeta}
	if err := g.targetClient.List(ctx, metadataList, opts...); err != nil {
		log.Error(err, "deletiongroups: error listing resources")
		return nil, fmt.Errorf("error listing resources for type %v: %w", typeMeta, err)
	}

	resourceList := convertList(metadataList.Items, func(item *metav1.PartialObjectMetadata) *managedresource.ManagedResourceStatus {
		return &managedresource.ManagedResourceStatus{
			Resource: corev1.ObjectReference{
				APIVersion: item.APIVersion,
				Kind:       item.Kind,
				Name:       item.GetName(),
				Namespace:  item.GetNamespace(),
			},
		}
	})

	return resourceList, nil
}

func convertList[E, F any](list []E, convertItem func(*E) F) []F {
	result := make([]F, len(list))
	for i := range list {
		result[i] = convertItem(&list[i])
	}
	return result
}

func (g *DeletionGroup) filterResourcesByNames(
	resources []*managedresource.ManagedResourceStatus,
	names []string,
) []*managedresource.ManagedResourceStatus {
	if len(names) == 0 {
		// no filter is provided; return complete list of resources
		return resources
	}

	// return list of resources whose name is contained in names.
	result := make([]*managedresource.ManagedResourceStatus, 0)
	for i := range resources {
		if slices.Contains(names, resources[i].Resource.Name) {
			result = append(result, resources[i])
		}
	}

	return result
}

// DELETE

func (g *DeletionGroup) Delete(ctx context.Context) (err error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	if err := ctx.Err(); err != nil {
		// Currently, only tests might cancel the context
		log.Info("context cancelled before processing deletiongroup", lc.KeyError, err)
		err = fmt.Errorf("context cancelled before processing deletiongroup: %w", err)
		return err
	}
	if err := g.interruptionChecker.Check(ctx); errors.Is(err, interruption.ErrInterruption) {
		log.Info("interruption before processing deletiongroup", lc.KeyError, err)
		err = fmt.Errorf("interruption before processing deletiongroup: %w", err)
		return err
	}

	var resources []*managedresource.ManagedResourceStatus
	if g.isDeleteAllResources() {
		resources, err = g.GetAllResources(ctx)
		if err != nil {
			return err
		}
	} else {
		resources = g.GetManagedResources()
	}

	for {
		resources, err = g.deleteResources(ctx, resources)
		if err != nil {
			return err
		}
		if len(resources) == 0 {
			// all resources are gone
			break
		}

		if ctx.Err() != nil {
			// Currently, only tests might cancel the context
			log.Info("context cancelled during processing deletiongroup", lc.KeyError, err)
			err = fmt.Errorf("context cancelled during processing deletiongroup: %w", err)
			return err
		}
		if err := g.interruptionChecker.Check(ctx); errors.Is(err, interruption.ErrInterruption) {
			log.Info("interruption during processing deletiongroup", lc.KeyError, err)
			err = fmt.Errorf("interruption during processing deletiongroup: %w", err)
			return err
		}

		time.Sleep(time.Second)
	}

	return nil
}

func (g *DeletionGroup) deleteResources(ctx context.Context, resources []*managedresource.ManagedResourceStatus) ([]*managedresource.ManagedResourceStatus, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	remainingResources := []*managedresource.ManagedResourceStatus{}

	for _, res := range resources {
		if _, err := timeout.TimeoutExceeded(ctx, g.deployItem, TimeoutCheckpointDeployerDeleteResources); err != nil {
			log.Info("timeout during processing deletiongroup", lc.KeyError, err)
			return nil, fmt.Errorf("timeout during processing deletiongroup: %w", err)
		}

		stillExists := g.deleteResource(ctx, res)
		if stillExists {
			remainingResources = append(remainingResources, res)
		}
	}

	return remainingResources, nil
}

func (g *DeletionGroup) deleteResource(ctx context.Context, res *managedresource.ManagedResourceStatus) (exists bool) {
	obj := kutil.ObjectFromCoreObjectReference(&res.Resource)
	key := client.ObjectKeyFromObject(obj)
	log, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyResource, key.String(),
		lc.KeyResourceVersion, res.Resource.APIVersion,
		lc.KeyResourceKind, res.Resource.Kind,
	)

	if err := g.targetClient.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
			// This handles two cases:
			// 1. the resource is already deleted
			// 2. the resource is a custom resource and its CRD is already deleted (and the resourse itself thus too)
			return false
		}

		log.Info("deletiongroups: error deleting resource", lc.KeyError, err)
		return true
	}

	if g.isForceDelete() {
		if err := g.targetClient.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
				return false
			}

			log.Info("deletiongroups: error fetching resource", lc.KeyError, err)
			return true
		}

		if len(obj.GetFinalizers()) > 0 {
			obj.SetFinalizers(nil)
			if err := g.targetClient.Update(ctx, obj); err != nil {
				if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
					return false
				}

				log.Info("deletiongroups: error removing finalizer from resource", lc.KeyError, err)
				return true
			}
		}
	}

	return true
}
