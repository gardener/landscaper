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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserrors "github.com/gardener/landscaper/apis/errors"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils"
)

const isEnabled = true

const (
	keyMyPodName = "myPodName"
	keyNamespace = "lockNamespace"

	kindDeployItem         = "DeployItem"
	kindSingletonByTimeout = "SingletonByTimeout"

	cleanupInterval = 2 * time.Minute
)

type Locker struct {
	lsClient   client.Client
	hostClient client.Client
}

func NewLocker(lsClient, hostClient client.Client) *Locker {
	return &Locker{
		lsClient:   lsClient,
		hostClient: hostClient,
	}
}

func (l *Locker) LockSingletonByTimeout(ctx context.Context, namespace, name string) (*lsv1alpha1.SyncObject, lserrors.LsError) {
	if !isEnabled {
		return &lsv1alpha1.SyncObject{}, nil
	}

	return nil, nil
}

func (l *Locker) LockDI(ctx context.Context, obj *lsv1alpha1.DeployItem) (*lsv1alpha1.SyncObject, lserrors.LsError) {
	return l.lock(ctx, obj, kindDeployItem)
}

func (l *Locker) lock(ctx context.Context, obj client.Object, kind string) (*lsv1alpha1.SyncObject, lserrors.LsError) {
	if !isEnabled {
		return &lsv1alpha1.SyncObject{}, nil
	}

	op := "Locker.Lock"

	log, ctx := logging.FromContextOrNew(ctx, nil, keyMyPodName, utils.GetCurrentPodName())

	syncObject, err := l.getSyncObject(ctx, obj.GetNamespace(), string(obj.GetUID()))
	if err != nil {
		lsError := lserrors.NewWrappedError(err, op, "resolveSecret", "error getting syncobject")
		return nil, lsError
	}

	if syncObject == nil {
		// the object is not yet locked; try to lock it
		syncObject = l.newSyncObject(obj, kind)
		err = l.lsClient.Create(ctx, syncObject)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// someone else was faster
				return nil, nil
			}

			msg := "locker: unable to create syncobject"
			log.Error(err, msg)
			lsError := lserrors.NewWrappedError(err, op, "createSyncObject", msg)
			return nil, lsError
		}

		// we have locked the object
		log.Info("locker: lock created")
		return syncObject, nil
	}

	if syncObject.Spec.PodName == utils.GetCurrentPodName() {
		log.Info("locker: object is already locked by this pod")
		return syncObject, nil
	}

	// check if syncObject.Spec.PodName contains the name of an existing pod
	// returns also false if syncObject.Spec.PodName is empty
	podExists, err := l.existsPod(ctx, syncObject.Spec.PodName)
	if err != nil {
		lsError := lserrors.NewWrappedError(err, op, "checkPodExists", "error checking if pod exists")
		return nil, lsError
	}

	if podExists {
		// the object is locked by another pod which indeed exists
		return nil, nil
	}

	// now we can try to take over the lock
	syncObject.Spec.PodName = utils.GetCurrentPodName()
	syncObject.Spec.LastUpdateTime = metav1.Now()
	if err := l.lsClient.Update(ctx, syncObject); err != nil {
		if apierrors.IsConflict(err) {
			// another pod has taken over the lock faster
			return nil, nil
		}

		msg := "locker: unable to take over lock"
		log.Error(err, msg)
		lsError := lserrors.NewWrappedError(err, op, "takeOverLock", msg)
		return nil, lsError
	}

	log.Info("locker: lock taken over")
	return syncObject, nil
}

func (l *Locker) Unlock(ctx context.Context, syncObject *lsv1alpha1.SyncObject) {
	if !isEnabled {
		return
	}

	log, ctx := logging.FromContextOrNew(ctx, nil, keyMyPodName, utils.GetCurrentPodName())

	syncObject.Spec.PodName = ""
	syncObject.Spec.LastUpdateTime = metav1.Now()
	if err := l.lsClient.Update(ctx, syncObject); err != nil {
		log.Error(err, "locker: unable to unlock syncobject")
		return
	}

	log.Info("locker: object unlocked")
}

func (l *Locker) NotLockedResult() (reconcile.Result, error) {
	return reconcile.Result{RequeueAfter: 3 * time.Minute}, nil
}

func (l *Locker) newSyncObject(obj client.Object, kind string) *lsv1alpha1.SyncObject {
	return &lsv1alpha1.SyncObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(obj.GetUID()),
			Namespace: obj.GetNamespace(),
		},
		Spec: lsv1alpha1.SyncObjectSpec{
			PodName:        utils.GetCurrentPodName(),
			Kind:           kind,
			Name:           obj.GetName(),
			LastUpdateTime: metav1.Now(),
		},
	}
}

func (l *Locker) getSyncObject(ctx context.Context, namespace, name string) (*lsv1alpha1.SyncObject, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	syncObjectKey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	syncObject := &lsv1alpha1.SyncObject{}
	if err := l.lsClient.Get(ctx, syncObjectKey, syncObject); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		log.Error(err, "locker: unable to get syncobject")
		return nil, fmt.Errorf("locker: unable to get syncobject: %w", err)
	}

	return syncObject, nil
}

func (l *Locker) existsPod(ctx context.Context, podName string) (bool, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	if podName == "" {
		return false, nil
	}

	podKey := client.ObjectKey{
		Namespace: utils.GetCurrentPodNamespace(),
		Name:      podName,
	}
	pod := &v1.Pod{}
	if err := l.hostClient.Get(ctx, podKey, pod); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		log.Error(err, "locker: unable to get pod")
		return false, fmt.Errorf("locker: unable to get pod %s: %w", podName, err)
	}

	return true, nil
}

func (l *Locker) StartPeriodicalSyncObjectCleanup(ctx context.Context, logger logging.Logger) {
	log := logger.WithName("syncobject-cleanup")
	ctx = logging.NewContext(ctx, log)

	log.Info("locker: starting periodical syncobject cleanup")

	startDelay := time.Duration(rand.Float64() * float64(cleanupInterval))
	time.Sleep(startDelay)

	wait.UntilWithContext(ctx, l.cleanupSyncObjects, cleanupInterval)
}

func (l *Locker) cleanupSyncObjects(ctx context.Context) {
	log, ctx := logging.FromContextOrNew(ctx, nil)
	log.Info("starting syncobject cleanup")

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

func (l *Locker) cleanupSyncObject(ctx context.Context, syncObject *lsv1alpha1.SyncObject) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

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

	log.Info("locker: cleanup of syncobject done")
}

func (l *Locker) existsResource(ctx context.Context, syncObject *lsv1alpha1.SyncObject) (bool, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	var resource client.Object

	switch syncObject.Spec.Kind {
	case kindDeployItem:
		resource = &lsv1alpha1.DeployItem{}
	default:
		log.Error(nil, "locker: unsupported kind", lc.KeyResourceKind, syncObject.Spec.Kind)
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

	return syncObject.GetName() == string(resource.GetUID()), nil
}
