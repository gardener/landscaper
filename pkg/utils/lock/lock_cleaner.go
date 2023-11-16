package lock

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils"
)

const (
	cleanupInterval = 3 * time.Hour
)

type LockCleaner struct {
	writer       *read_write_layer.Writer
	lsReadClient client.Client
}

func NewLockCleaner(lsClient client.Client) *LockCleaner {
	writer := read_write_layer.NewWriter(lsClient)

	return &LockCleaner{
		writer:       writer,
		lsReadClient: lsClient,
	}
}

func (l *LockCleaner) StartPeriodicalSyncObjectCleanup(ctx context.Context, logger logging.Logger) {
	log := logger.WithName("syncobject-cleanup")
	ctx = logging.NewContext(ctx, log)

	log.Info("locker: starting periodical syncobject cleanup")

	startDelay := time.Duration(rand.Float64() * float64(cleanupInterval))
	time.Sleep(startDelay)

	wait.UntilWithContext(ctx, l.cleanupSyncObjects, cleanupInterval)
}

func (l *LockCleaner) cleanupSyncObjects(ctx context.Context) {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Info("locker: starting syncobject cleanup")

	namespaces := &v1.NamespaceList{}
	if err := l.lsReadClient.List(ctx, namespaces); err != nil {
		log.Error(err, "locker: failed to list namespaces")
		return
	}

	for _, namespace := range namespaces.Items {
		syncObjects := &lsv1alpha1.SyncObjectList{}
		if err := read_write_layer.ListSyncObjects(ctx, l.lsReadClient, syncObjects, client.InNamespace(namespace.Name)); err != nil {
			log.Error(err, "locker: failed to list syncobjects in namespace", keyNamespace, namespace.Name)
			continue
		}

		for _, syncObject := range syncObjects.Items {
			l.cleanupSyncObject(ctx, &syncObject)
		}
	}
}

func (l *LockCleaner) cleanupSyncObject(ctx context.Context, syncObject *lsv1alpha1.SyncObject) {
	log, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyResource, client.ObjectKeyFromObject(syncObject).String(),
		lc.KeyResourceKind, syncObject.Spec.Kind)

	exists, err := l.existsResource(ctx, syncObject)
	if err != nil {
		return
	}

	if exists {
		return
	}

	if err := l.writer.DeleteSyncObject(ctx, read_write_layer.W000046, syncObject); err != nil {
		log.Error(err, "locker: cleanup of syncobject failed")
		return
	}

	log.Debug("locker: cleanup of syncobject done")
}

func (l *LockCleaner) existsResource(ctx context.Context, syncObject *lsv1alpha1.SyncObject) (bool, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	var resource *metav1.PartialObjectMetadata

	switch syncObject.Spec.Kind {
	case utils.InstallationKind:
		resource = utils.EmptyInstallationMetadata()
	case utils.ExecutionKind:
		resource = utils.EmptyExecutionMetadata()
	case utils.DeployItemKind:
		resource = utils.EmptyDeployItemMetadata()
	default:
		log.Error(nil, "locker: unsupported kind")
		return false, fmt.Errorf("locker: unsupported kind %s", syncObject.Spec.Kind)
	}

	resourceKey := client.ObjectKey{
		Namespace: syncObject.Namespace,
		Name:      syncObject.Spec.Name,
	}

	if err := l.lsReadClient.Get(ctx, resourceKey, resource); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		log.Error(err, "locker: unable to check existence of locked resource")
		return false, err
	}

	return syncObject.GetName() == getName(syncObject.Spec.Prefix, resource), nil
}
