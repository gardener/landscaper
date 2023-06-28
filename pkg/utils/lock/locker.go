package lock

import (
	"context"
	"fmt"
	"time"

	lserrors "github.com/gardener/landscaper/apis/errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"
)

const isEnabled = true

const (
	KeyMyPodName    = "myPodName"
	KeyOtherPodName = "otherPodName"
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

func (l *Locker) Lock(ctx context.Context, obj client.Object) (*lsv1alpha1.SyncObject, lserrors.LsError) {
	op := "Locker.Lock"

	if !isEnabled {
		return &lsv1alpha1.SyncObject{}, nil
	}

	log, ctx := logging.FromContextOrNew(ctx, nil, KeyMyPodName, utils.GetCurrentPodName())

	syncObject, err := l.getSyncObject(ctx, obj)
	if err != nil {
		lsError := lserrors.NewWrappedError(err, op, "resolveSecret", "error getting sync object")
		return nil, lsError
	}

	if syncObject == nil {
		// the object is not yet locked; try to lock it
		syncObject = l.newSyncObject(obj)
		err = l.lsClient.Create(ctx, syncObject)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// someone else was faster
				return nil, nil
			}

			msg := "locking: unable to create syncobject"
			log.Error(err, msg)
			lsError := lserrors.NewWrappedError(err, op, "createSyncObject", msg)
			return nil, lsError
		}

		// we have locked the object
		log.Info("locking: lock created")
		return syncObject, nil
	}

	if syncObject.Spec.PodName == utils.GetCurrentPodName() {
		log.Info("locking: object is already locked by this pod")
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

	log, ctx := logging.FromContextOrNew(ctx, nil, KeyMyPodName, utils.GetCurrentPodName())

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

func (l *Locker) newSyncObject(obj client.Object) *lsv1alpha1.SyncObject {
	return &lsv1alpha1.SyncObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(obj.GetUID()),
			Namespace: obj.GetNamespace(),
		},
		Spec: lsv1alpha1.SyncObjectSpec{
			PodName:        utils.GetCurrentPodName(),
			Kind:           obj.GetObjectKind().GroupVersionKind().Kind,
			Name:           obj.GetName(),
			LastUpdateTime: metav1.Now(),
		},
	}
}

func (l *Locker) getSyncObject(ctx context.Context, obj client.Object) (*lsv1alpha1.SyncObject, error) {
	log, ctx := logging.FromContextOrNew(ctx, nil)

	syncObjectKey := client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      string(obj.GetUID()),
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
