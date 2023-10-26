// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"fmt"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lserror "github.com/gardener/landscaper/apis/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	StatusReady    StatusType = "Ready"
	StatusNotReady StatusType = "NotReady"
	StatusUnknown  StatusType = "Unknown"
)

// StatusType defines the value of a Status.
type StatusType string

// checkObjectFunc is a function to perform the actual readiness check
type checkObjectFunc func(*unstructured.Unstructured) error

type ObjectsToWatchFunc func() ([]*unstructured.Unstructured, error)

// InterruptionChecker is the interface to check for interrupts during the readiness check.
type InterruptionChecker interface {
	Check(context.Context) error
}

// WaitForObjectsReady waits for objects to be heatlhy and
// returns an error if all the objects are not ready after the timeout.
func WaitForObjectsReady(ctx context.Context, timeout time.Duration, kubeClient client.Client,
	getObjects ObjectsToWatchFunc, fn checkObjectFunc, interruptionChecker InterruptionChecker, operation string) error {
	var (
		try     int32 = 1
		err     error
		objects []*unstructured.Unstructured
	)
	log, ctx := logging.FromContextOrNew(ctx, nil)

	checkpoint := fmt.Sprintf("deployer: during readiness check")
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		log.Debug("Wait until resources are ready", "try", try)
		try++

		var err error
		if err = interruptionChecker.Check(ctx); err != nil {
			return false, err
		}

		objects, err = getObjects()
		if err != nil {
			if IsRecoverableError(err) {
				log.Info("WaitForObjectsReady: failed getObjects: " + err.Error())
				return false, nil
			} else {
				log.Error(err, "WaitForObjectsReady: failed getObjects: "+err.Error())
				return false, nil
			}
		}

		for _, obj := range objects {
			checkpoint = fmt.Sprintf("deployer: during readiness check - resource %s/%s of type %s",
				obj.GetNamespace(), obj.GetName(), obj.GetKind())

			if err = IsObjectReady(ctx, kubeClient, obj, fn); err != nil {
				if IsRecoverableError(err) {
					log.Info(fmt.Sprintf("WaitForObjectsReady: resource %s/%s of type %s is not ready",
						obj.GetNamespace(), obj.GetName(), obj.GetKind()))
					return false, nil
				} else {
					log.Error(err, fmt.Sprintf("WaitForObjectsReady: resource %s/%s of type %s is not ready",
						obj.GetNamespace(), obj.GetName(), obj.GetKind()))
					return false, nil
				}
			}
		}

		return true, nil
	})

	if wait.Interrupted(err) {
		msg := fmt.Sprintf("timeout at: %q", checkpoint)
		return lserror.NewWrappedError(err, "WaitForObjectsReady", lsv1alpha1.ProgressingTimeoutReason, msg, lsv1alpha1.ErrorTimeout)
	}

	return err
}

// RecoverableError defines an error that occurs during the readiness check,
// but is recoverable. Means, the error could be sporadic and therefore
// the readiness check is not interrupted.
type RecoverableError struct {
	err error
}

// Error implements the go error interface.
func (e *RecoverableError) Error() string {
	return fmt.Sprintf("recoverable error: %s", e.err.Error())
}

func NewRecoverableError(err error) *RecoverableError {
	return &RecoverableError{
		err: err,
	}
}

// ObjectNotReadyError holds information about an unready object
// and implements the go error interface.
// This is a subtype of RecoverableError.
type ObjectNotReadyError struct {
	objectGVK       string
	objectName      string
	objectNamespace string
	err             error
}

// Error implements the go error interface.
func (e *ObjectNotReadyError) Error() string {
	return fmt.Sprintf("%s %s/%s is not ready: %s",
		e.objectGVK,
		e.objectName,
		e.objectNamespace,
		e.err.Error())
}

func NewObjectNotReadyError(u *unstructured.Unstructured, err error) *ObjectNotReadyError {
	return &ObjectNotReadyError{
		objectGVK:       u.GroupVersionKind().String(),
		objectName:      u.GetName(),
		objectNamespace: u.GetNamespace(),
		err:             err,
	}
}

func IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	switch err.(type) {
	case *RecoverableError:
		return true
	case *ObjectNotReadyError:
		return true
	default:
		return false
	}
}

// IsObjectReady gets an updated version of an object and checks if it is ready.
func IsObjectReady(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured,
	checkObject checkObjectFunc) error {
	objLog, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyGroupVersionKind, obj.GroupVersionKind().String(),
		lc.KeyResource, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()).String())

	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, obj); err != nil {
		objLog.Debug("Resource status", lc.KeyStatus, StatusUnknown)
		return NewRecoverableError(fmt.Errorf("unable to get %s %s/%s: %w",
			obj.GroupVersionKind().String(),
			obj.GetName(), obj.GetNamespace(),
			err))
	}

	objLog.Debug("Getting resource status")
	if err := checkObject(obj); err != nil {
		objLog.Debug("Resource status", lc.KeyStatus, StatusNotReady, "reason", err.Error())
		return err
	}

	objLog.Debug("Resource status", lc.KeyStatus, StatusReady)

	return nil
}
