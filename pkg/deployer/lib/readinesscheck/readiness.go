// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
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

// isCheckRelevantFunc is a function that determines the check relevance of an object
type isCheckRelevantFunc func(*unstructured.Unstructured) bool

// WaitForObjectsReady waits for objects to be heatlhy and
// returns an error if all the objects are not ready after the timeout.
func WaitForObjectsReady(ctx context.Context, timeout time.Duration, kubeClient client.Client,
	objects []*unstructured.Unstructured, fn checkObjectFunc, isCheckRelevant isCheckRelevantFunc, failOnMissingObject bool) error {
	var (
		wg  sync.WaitGroup
		try int32 = 1

		// allErrs contains all the errors not related to the readiness of objects.
		allErrs []error
		// notReadyErrs contains all the errors related to the readiness of objects.
		notReadyErrs []error
	)
	log, ctx := logging.FromContextOrNew(ctx, nil)

	_ = wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		log.Debug("Wait until resources are ready", "try", try)
		try++

		allErrs = nil
		notReadyErrs = nil
		for _, obj := range objects {
			wg.Add(1)
			go func(obj *unstructured.Unstructured) {
				defer wg.Done()

				if err := IsObjectReady(ctx, kubeClient, obj, fn, isCheckRelevant, failOnMissingObject); err != nil {
					switch err.(type) {
					case *ObjectNotReadyError:
						notReadyErrs = append(notReadyErrs, err)
					default:
						allErrs = append(allErrs, err)
					}
				}
			}(obj)
		}
		wg.Wait()

		if len(allErrs) > 0 {
			return false, apimacherrors.NewAggregate(allErrs)
		}
		if len(notReadyErrs) > 0 {
			return false, nil
		}

		return true, nil
	})

	if len(allErrs) > 0 {
		return apimacherrors.NewAggregate(allErrs)
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

// IsObjectReady gets an updated version of an object and checks if it is ready.
func IsObjectReady(ctx context.Context, kubeClient client.Client, obj *unstructured.Unstructured,
	checkObject checkObjectFunc, isCheckRelevant isCheckRelevantFunc, failOnMissingObject bool) error {
	objLog, ctx := logging.FromContextOrNew(ctx, nil,
		lc.KeyGroupVersionKind, obj.GroupVersionKind().String(),
		lc.KeyResource, kutil.ObjectKey(obj.GetName(), obj.GetNamespace()).String())

	if !isCheckRelevant(obj) && !failOnMissingObject {
		return nil
	}

	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, obj); err != nil {
		if !errors.IsNotFound(err) || isCheckRelevant(obj) || failOnMissingObject {
			objLog.Debug("Resource status", lc.KeyStatus, StatusUnknown)
			return fmt.Errorf("unable to get %s %s/%s: %w",
				obj.GroupVersionKind().String(),
				obj.GetName(), obj.GetNamespace(),
				err)
		}
	}

	objLog.Debug("Getting resource status")
	if err := checkObject(obj); err != nil {
		objLog.Debug("Resource status", lc.KeyStatus, StatusNotReady)
		return &ObjectNotReadyError{
			objectGVK:       obj.GroupVersionKind().String(),
			objectName:      obj.GetName(),
			objectNamespace: obj.GetNamespace(),
			err:             err,
		}
	}

	objLog.Debug("Resource status", lc.KeyStatus, StatusReady)

	return nil
}
