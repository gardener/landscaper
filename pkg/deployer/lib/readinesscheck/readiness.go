// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
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

// WaitForObjectsReady waits for objects to be heatlhy and
// returns an error if all the objects are not ready after the timeout.
func WaitForObjectsReady(ctx context.Context, timeout time.Duration, log logr.Logger, kubeClient client.Client, objects []*unstructured.Unstructured, fn checkObjectFunc) error {
	var (
		wg  sync.WaitGroup
		try int32 = 1

		// allErrs contains all the errors not related to the readiness of objects.
		allErrs []error
		// notReadyErrs contains all the errors related to the readiness of objects.
		notReadyErrs []error
	)

	_ = wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		log.V(3).Info("wait resources ready", "try", try)
		try++

		allErrs = nil
		notReadyErrs = nil
		for _, obj := range objects {
			wg.Add(1)
			go func(obj *unstructured.Unstructured) {
				defer wg.Done()

				if err := IsObjectReady(ctx, log, kubeClient, obj, fn); err != nil {
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
func IsObjectReady(ctx context.Context, log logr.Logger, kubeClient client.Client, obj *unstructured.Unstructured, checkObject checkObjectFunc) error {
	objLog := log.WithValues(
		"object", obj.GroupVersionKind().String(),
		"resource", kutil.ObjectKey(obj.GetName(), obj.GetNamespace()))

	key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
	if err := kubeClient.Get(ctx, key, obj); err != nil {
		objLog.V(3).Info("resource status", "status", StatusUnknown)
		return fmt.Errorf("unable to get %s %s/%s: %w",
			obj.GroupVersionKind().String(),
			obj.GetName(), obj.GetNamespace(),
			err)
	}

	objLog.V(3).Info("get resource status")
	if err := checkObject(obj); err != nil {
		objLog.V(3).Info("resource status", "status", StatusNotReady)
		return &ObjectNotReadyError{
			objectGVK:       obj.GroupVersionKind().String(),
			objectName:      obj.GetName(),
			objectNamespace: obj.GetNamespace(),
			err:             err,
		}
	}

	objLog.V(3).Info("resource status", "status", StatusReady)

	return nil
}
