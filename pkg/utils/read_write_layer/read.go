package read_write_layer

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// read methods get, list

// read methods for sync objects
func GetSyncObject(ctx context.Context, c client.Reader, key client.ObjectKey, syncObject *lsv1alpha1.SyncObject, readID ReadID) error {
	return get(ctx, c, key, syncObject)
}

func ListSyncObjects(ctx context.Context, c client.Reader, syncObjects *lsv1alpha1.SyncObjectList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, syncObjects, opts...)
}

// read methods for installations
func GetInstallation(ctx context.Context, c client.Reader, key client.ObjectKey, installation *lsv1alpha1.Installation, readID ReadID) error {
	return get(ctx, c, key, installation)
}

func ListInstallations(ctx context.Context, c client.Reader, installations *lsv1alpha1.InstallationList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, installations, opts...)
}

// read methods for executions
func GetExecution(ctx context.Context, c client.Reader, key client.ObjectKey, execution *lsv1alpha1.Execution, readID ReadID) error {
	return get(ctx, c, key, execution)
}

// read methods for deploy items
func GetDeployItem(ctx context.Context, c client.Reader, key client.ObjectKey, deployItem *lsv1alpha1.DeployItem, readID ReadID) error {
	return get(ctx, c, key, deployItem)
}

func ListDeployItems(ctx context.Context, c client.Reader, deployItems *lsv1alpha1.DeployItemList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, deployItems, opts...)
}

// read methods for target

func GetTarget(ctx context.Context, c client.Reader, key client.ObjectKey, target *lsv1alpha1.Target, readID ReadID) error {
	return get(ctx, c, key, target)
}

// read methods for secret

func GetSecret(ctx context.Context, c client.Reader, key client.ObjectKey, secret *v1.Secret, readID ReadID) error {
	return get(ctx, c, key, secret)
}

// read methods for secret

func GetHealthCheck(ctx context.Context, c client.Reader, key client.ObjectKey, health *lsv1alpha1.LsHealthCheck, readID ReadID) error {
	return get(ctx, c, key, health)
}

// read methods for env

func GetEnv(ctx context.Context, c client.Reader, key client.ObjectKey, env *lsv1alpha1.Environment, readID ReadID) error {
	return get(ctx, c, key, env)
}

// read methods for object
func GetObject(ctx context.Context, c client.Reader, key client.ObjectKey, object client.Object, readID ReadID) error {
	return get(ctx, c, key, object)
}

// read methods for pods
func GetPod(ctx context.Context, c client.Reader, key client.ObjectKey, pod *v1.Pod, readID ReadID) error {
	return get(ctx, c, key, pod)
}

// read methods for metadata
func GetMetaData(ctx context.Context, c client.Reader, key client.ObjectKey, metadata *v12.PartialObjectMetadata, readID ReadID) error {
	return get(ctx, c, key, metadata)
}

// read methods for context
func GetContext(ctx context.Context, c client.Reader, key client.ObjectKey, lsContext *lsv1alpha1.Context, readID ReadID) error {
	return get(ctx, c, key, lsContext)
}

// read methods for unstructured
func GetUnstructured(ctx context.Context, c client.Reader, key client.ObjectKey, unstruc *unstructured.Unstructured, readID ReadID) error {
	return get(ctx, c, key, unstruc)
}

// basic functions
func get(ctx context.Context, c client.Reader, key client.ObjectKey, object client.Object) error {
	log, ctx := logging.FromContextOrNew(ctx, nil, keyFetchedResource, fmt.Sprintf("%s/%s", key.Namespace, key.Name))
	log.Debug("ReadLayer get")

	return c.Get(ctx, key, object)
}

func list(ctx context.Context, c client.Reader, objects client.ObjectList, opts ...client.ListOption) error {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Debug("ReadLayer list")

	return c.List(ctx, objects, opts...)
}
