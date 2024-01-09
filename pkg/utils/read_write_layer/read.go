package read_write_layer

import (
	"context"
	"fmt"
	"strconv"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

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
	return get(ctx, c, key, syncObject, readID, "syncObject")
}

func ListSyncObjects(ctx context.Context, c client.Reader, syncObjects *lsv1alpha1.SyncObjectList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, syncObjects, readID, "syncObjects", opts...)
}

// read methods for installations
func GetInstallation(ctx context.Context, c client.Reader, key client.ObjectKey, installation *lsv1alpha1.Installation, readID ReadID) error {
	return get(ctx, c, key, installation, readID, "installation")
}

func ListInstallations(ctx context.Context, c client.Reader, installations *lsv1alpha1.InstallationList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, installations, readID, "installations", opts...)
}

// read methods for executions
func GetExecution(ctx context.Context, c client.Reader, key client.ObjectKey, execution *lsv1alpha1.Execution, readID ReadID) error {
	return get(ctx, c, key, execution, readID, "execution")
}

// read methods for deploy items
func GetDeployItem(ctx context.Context, c client.Reader, key client.ObjectKey, deployItem *lsv1alpha1.DeployItem, readID ReadID) error {
	return get(ctx, c, key, deployItem, readID, "deployItem")
}

func ListManagedDeployItems(ctx context.Context, c client.Reader, execKey client.ObjectKey, readID ReadID) (*lsv1alpha1.DeployItemList, error) {
	deployItemList := &lsv1alpha1.DeployItemList{}
	// todo: maybe use name and namespace
	if err := ListDeployItems(ctx, c, deployItemList, readID,
		client.MatchingLabels{lsv1alpha1.ExecutionManagedByLabel: execKey.Name},
		client.InNamespace(execKey.Namespace)); err != nil {
		return nil, err
	}
	return deployItemList, nil
}

func ListDeployItems(ctx context.Context, c client.Reader, deployItems *lsv1alpha1.DeployItemList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, deployItems, readID, "deployItems", opts...)
}

// read methods for target

func GetTarget(ctx context.Context, c client.Reader, key client.ObjectKey, target *lsv1alpha1.Target, readID ReadID) error {
	return get(ctx, c, key, target, readID, "target")
}

func ListTargets(ctx context.Context, c client.Reader, targets *lsv1alpha1.TargetList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, targets, readID, "targets", opts...)
}

// read methods for data objects

func ListDataObjects(ctx context.Context, c client.Reader, dataObjects *lsv1alpha1.DataObjectList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, dataObjects, readID, "dataObjects", opts...)
}

// read methods for targetsync

func ListTargetSyncs(ctx context.Context, c client.Reader, targetSyncs *lsv1alpha1.TargetSyncList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, targetSyncs, readID, "targetSyncs", opts...)
}

// read methods for secret

func GetSecret(ctx context.Context, c client.Reader, key client.ObjectKey, secret *v1.Secret, readID ReadID) error {
	return get(ctx, c, key, secret, readID, "secret")
}

func ListSecrets(ctx context.Context, c client.Reader, secrets *v1.SecretList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, secrets, readID, "secrets", opts...)
}

// read methods for health checks

func GetHealthCheck(ctx context.Context, c client.Reader, key client.ObjectKey, health *lsv1alpha1.LsHealthCheck, readID ReadID) error {
	return get(ctx, c, key, health, readID, "healthCheck")
}

func ListHealthChecks(ctx context.Context, c client.Reader, healthChecks *lsv1alpha1.LsHealthCheckList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, healthChecks, readID, "healthChecks", opts...)
}

// read methods for context
func GetContext(ctx context.Context, c client.Reader, key client.ObjectKey, lsContext *lsv1alpha1.Context, readID ReadID) error {
	return get(ctx, c, key, lsContext, readID, "context")
}

// read methods for env

func GetEnv(ctx context.Context, c client.Reader, key client.ObjectKey, env *lsv1alpha1.Environment, readID ReadID) error {
	return get(ctx, c, key, env, readID, "env")
}

// read methods for object
func GetObject(ctx context.Context, c client.Reader, key client.ObjectKey, object client.Object, readID ReadID) error {
	return get(ctx, c, key, object, readID, "object")
}

// read methods for pods
func GetPod(ctx context.Context, c client.Reader, key client.ObjectKey, pod *v1.Pod, readID ReadID) error {
	return get(ctx, c, key, pod, readID, "pod")
}

func ListPods(ctx context.Context, c client.Reader, pods *v1.PodList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, pods, readID, "pods", opts...)
}

// read methods for namespaces
func ListNamespaces(ctx context.Context, c client.Reader, namespaces *v1.NamespaceList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, namespaces, readID, "namespaces", opts...)
}

// read methods for metadata
func GetMetaData(ctx context.Context, c client.Reader, key client.ObjectKey, metadata *v12.PartialObjectMetadata, readID ReadID) error {
	return get(ctx, c, key, metadata, readID, "metadata")
}

func ListMetaData(ctx context.Context, c client.Reader, metadataList *v12.PartialObjectMetadataList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, metadataList, readID, "metadataList", opts...)
}

// read methods for unstructured
func GetUnstructured(ctx context.Context, c client.Reader, key client.ObjectKey, unstruc *unstructured.Unstructured, readID ReadID) error {
	return get(ctx, c, key, unstruc, readID, "unstructured")
}

func ListUnstructured(ctx context.Context, c client.Reader, unstrucs *unstructured.UnstructuredList, readID ReadID, opts ...client.ListOption) error {
	return list(ctx, c, unstrucs, readID, "unstructureds", opts...)
}

// basic functions
func get(ctx context.Context, c client.Reader, key client.ObjectKey, object client.Object, readID ReadID, msg string) error {
	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", key.Namespace, key.Name),
		lc.KeyReadID, readID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.Get(ctx, key, object)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug("ReadLayer get: " + msg)
	}

	return err
}

func list(ctx context.Context, c client.Reader, objects client.ObjectList, readID ReadID, msg string, opts ...client.ListOption) error {
	log, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyReadID, readID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.List(ctx, objects, opts...)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug("ReadLayer list: " + msg)
	}

	return err
}
