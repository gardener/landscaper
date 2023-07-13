package lock

import (
	"context"
	"fmt"
	"math/rand"
	"time"

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

type LockCleaner struct {
	lsClient   client.Client
	hostClient client.Client
}

func NewLockCleaner(lsClient, hostClient client.Client) *LockCleaner {
	return &LockCleaner{
		lsClient:   lsClient,
		hostClient: hostClient,
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
	if err := l.lsClient.List(ctx, namespaces); err != nil {
		log.Error(err, "locker: failed to list namespaces")
		return
	}

	for _, namespace := range namespaces.Items {
		syncObjects := &lsv1alpha1.SyncObjectList{}
		if err := l.lsClient.List(ctx, syncObjects, client.InNamespace(namespace.Name)); err != nil {
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

	if err := l.lsClient.Delete(ctx, syncObject); err != nil {
		log.Error(err, "locker: cleanup of syncobject failed")
		return
	}

	log.Debug("locker: cleanup of syncobject done")
}

func (l *LockCleaner) existsResource(ctx context.Context, syncObject *lsv1alpha1.SyncObject) (bool, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	var resource *metav1.PartialObjectMetadata

	switch syncObject.Spec.Kind {
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

	if err := l.hostClient.Get(ctx, resourceKey, resource); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		log.Error(err, "locker: unable to check existence of locked resource")
		return false, err
	}

	return syncObject.GetName() == getName(syncObject.Spec.Prefix, resource), nil
}
