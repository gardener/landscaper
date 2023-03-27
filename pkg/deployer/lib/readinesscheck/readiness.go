// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib"
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

// WaitForObjectsReady waits for objects to be heatlhy and
// returns an error if all the objects are not ready after the timeout.
func WaitForObjectsReady(ctx context.Context, timeout time.Duration, kubeClient client.Client,
	getObjects ObjectsToWatchFunc, fn checkObjectFunc, interruptionChecker *lib.InterruptionChecker) error {
	var (
		wg  sync.WaitGroup
		try int32 = 1

		// notReadyErrs contains all the errors related to the readiness of objects.
		notReadyErrs []error
		// allErrs contains all the errors not related to the readiness of objects.
		otherErrs []error
	)
	log, ctx := logging.FromContextOrNew(ctx, nil)

	_ = wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		log.Debug("Wait until resources are ready", "try", try)
		try++

		if err := interruptionChecker.Check(ctx); err != nil {
			return false, err
		}

		objects, err := getObjects()
		if err != nil {
			switch err.(type) {
			case *ObjectNotReadyError:
				return false, nil
			default:
				return false, err
			}
		}

		allErrors := make([]error, len(objects))

		for i, obj := range objects {
			wg.Add(1)
			go func(obj *unstructured.Unstructured, i int, allErrors []error) {
				defer wg.Done()

				if err := IsObjectReady(ctx, kubeClient, obj, fn); err != nil {
					allErrors[i] = err
				}
			}(obj, i, allErrors)
		}
		wg.Wait()

		otherErrs = nil
		notReadyErrs = nil

		for _, err := range allErrors {
			if err != nil {
				switch err.(type) {
				case *ObjectNotReadyError:
					notReadyErrs = append(notReadyErrs, err)
				default:
					otherErrs = append(otherErrs, err)
				}
			}
		}

		if len(otherErrs) > 0 {
			return false, apimacherrors.NewAggregate(otherErrs)
		}
		if len(notReadyErrs) > 0 {
			return false, nil
		}

		return true, nil
	})

	if len(otherErrs) > 0 {
		return apimacherrors.NewAggregate(otherErrs)
	}
	if len(notReadyErrs) > 0 {
		return apimacherrors.NewAggregate(notReadyErrs)
	}

	return nil
}

// ObjectNotReadyError holds information about an unready object
// and implements the go error interface.
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

// IsObjectReady gets an updated version of an object and checks if it is ready.
func IsObjectReady(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured,
	checkObject checkObjectFunc) error {
	objLog, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyGroupVersionKind, obj.GroupVersionKind().String(),
		lc.KeyResource, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()).String())

	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, obj); err != nil {
		objLog.Debug("Resource status", lc.KeyStatus, StatusUnknown)
		return fmt.Errorf("unable to get %s %s/%s: %w",
			obj.GroupVersionKind().String(),
			obj.GetName(), obj.GetNamespace(),
			err)
	}

	objLog.Debug("Getting resource status")
	if err := checkObject(obj); err != nil {
		objLog.Debug("Resource status", lc.KeyStatus, StatusNotReady, "reason", err.Error())
		return err
	}

	objLog.Debug("Resource status", lc.KeyStatus, StatusReady)

	return nil
}
