// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthcheck

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
	StatusHealthy    StatusType = "Healthy"
	StatusNotHealthy StatusType = "NotHealthy"
	StatusUnknown    StatusType = "Unknown"
)

// StatusType defines the value of a Status.
type StatusType string

// checkObjectFunc is a function to be perform the actual health checks
type checkObjectFunc func(*unstructured.Unstructured) error

// WaitForObjectsHealthy waits for objects to be heatlhy and
// returns an error if all the objects are not healthy after the timeout.
func WaitForObjectsHealthy(ctx context.Context, timeout time.Duration, log logr.Logger, kubeClient client.Client, objects []*unstructured.Unstructured, fn checkObjectFunc) error {
	var (
		wg  sync.WaitGroup
		try int32 = 1

		// allErrs contains all the errors not related to the healthiness of objects.
		allErrs []error
		// notHealthyErrs contains all the errors related to the healthiness of objects.
		notHealthyErrs []error
	)

	_ = wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		log.V(3).Info("wait resources healthy", "try", try)
		try++

		allErrs = nil
		notHealthyErrs = nil
		for _, obj := range objects {
			wg.Add(1)
			go func(obj *unstructured.Unstructured) {
				defer wg.Done()

				if err := IsObjectHealthy(ctx, log, kubeClient, obj, fn); err != nil {
					switch err.(type) {
					case *ObjectNotHealthyError:
						notHealthyErrs = append(notHealthyErrs, err)
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
		if len(notHealthyErrs) > 0 {
			return false, nil
		}

		return true, nil
	})

	if len(allErrs) > 0 {
		return apimacherrors.NewAggregate(allErrs)
	}
	if len(notHealthyErrs) > 0 {
		return apimacherrors.NewAggregate(notHealthyErrs)
	}

	return nil
}

// ObjectNotHealthyError holds information about an unhealthy object
// and implements the go error interface.
type ObjectNotHealthyError struct {
	objectGVK       string
	objectName      string
	objectNamespace string
	err             error
}

// Error implements the go error interface.
func (e *ObjectNotHealthyError) Error() string {
	return fmt.Sprintf("%s %s/%s is not healthy: %s",
		e.objectGVK,
		e.objectName,
		e.objectNamespace,
		e.err.Error())
}

// IsObjectHealthy gets an updated version of an object and checks if it is healthy.
func IsObjectHealthy(ctx context.Context, log logr.Logger, kubeClient client.Client, obj *unstructured.Unstructured, checkObject checkObjectFunc) error {
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
		objLog.V(3).Info("resource status", "status", StatusNotHealthy)
		return &ObjectNotHealthyError{
			objectGVK:       obj.GroupVersionKind().String(),
			objectName:      obj.GetName(),
			objectNamespace: obj.GetNamespace(),
			err:             err,
		}
	}

	objLog.V(3).Info("resource status", "status", StatusHealthy)

	return nil
}
