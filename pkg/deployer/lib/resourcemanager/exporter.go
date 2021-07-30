// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
)

// ExporterOptions defines the options for the exporter.
type ExporterOptions struct {
	KubeClient     client.Client
	DefaultTimeout *time.Duration

	Objects managedresource.ManagedResourceStatusList
}

// Exporter defines the export of data from manifests.
type Exporter struct {
	log            logr.Logger
	kubeClient     client.Client
	defaultTimeout time.Duration

	objects managedresource.ManagedResourceStatusList
}

// NewExporter creates a new exporter.
func NewExporter(log logr.Logger, opts ExporterOptions) *Exporter {
	exporter := &Exporter{
		log:            log,
		kubeClient:     opts.KubeClient,
		defaultTimeout: 5 * time.Minute,
		objects:        opts.Objects,
	}
	if opts.DefaultTimeout != nil {
		exporter.defaultTimeout = *opts.DefaultTimeout
	}

	return exporter
}

// Export exports all keys that are defined in the exports definition.
func (e *Exporter) Export(ctx context.Context, exports *managedresource.Exports) (map[string]interface{}, error) {
	// first validate if referenced resource is managed.
	for _, export := range exports.Exports {
		if export.FromResource == nil {
			// ignore exports without from resource
			// this currently only used for helm values where no resource is needed.
			continue
		}

		if !e.resourceIsManaged(*export.FromResource) {
			return nil, fmt.Errorf("resource %s %s %s is not managed by the deployer", export.FromResource.APIVersion, export.FromResource.Name, export.FromResource.Namespace)
		}
	}
	var (
		allErrs     []error
		wg          sync.WaitGroup
		resultMutex sync.Mutex
		result      map[string]interface{}
	)
	for _, export := range exports.Exports {
		if export.FromResource == nil {
			// ignore exports without from resource
			// this currently only used for helm values where no resource is needed.
			continue
		}

		if exports.DefaultTimeout != nil {
			// use default timeout from exports
			export.Timeout = exports.DefaultTimeout
		} else if export.Timeout == nil {
			// use default timeout of deployer
			export.Timeout = &lsv1alpha1.Duration{
				Duration: e.defaultTimeout,
			}
		}
		wg.Add(1)
		go func(ctx context.Context, export managedresource.Export) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, export.Timeout.Duration)
			defer cancel()
			log := e.log.WithName("export").WithValues("key", export.Key)

			backoff := wait.Backoff{
				Jitter: 1.15,
				Steps:  math.MaxInt32,
				Cap:    export.Timeout.Duration,
			}
			var lastErr error
			if err := wait.ExponentialBackoffWithContext(ctx, backoff, func() (done bool, err error) {
				value, err := e.doExport(ctx, export)
				if err != nil {
					log.V(5).Info(err.Error())
					lastErr = err
					return false, nil
				}
				resultMutex.Lock()
				defer resultMutex.Unlock()
				result = utils.MergeMaps(result, value)
				return true, nil
			}); err != nil {
				allErrs = append(allErrs, err)
				if lastErr != nil {
					allErrs = append(allErrs, lastErr)
				}
			}

		}(ctx, export)
	}
	wg.Wait()
	if len(allErrs) != 0 {
		// todo: improve so that already retrieved values are persisted and only other ones have to be re-fetched.
		return nil, apimacherrors.NewAggregate(allErrs)
	}
	return result, nil
}

func (e *Exporter) doExport(ctx context.Context, export managedresource.Export) (map[string]interface{}, error) {
	// get resource from client
	obj := kutil.ObjectFromTypedObjectReference(export.FromResource)
	if err := e.kubeClient.Get(ctx, kutil.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, err
	}

	var val interface{}
	if err := jsonpath.GetValue(export.JSONPath, obj.Object, &val); err != nil {
		return nil, err
	}

	newValue, err := jsonpath.Construct(export.Key, val)
	if err != nil {
		return nil, err
	}
	return newValue, nil
}

func (e *Exporter) resourceIsManaged(res lsv1alpha1.TypedObjectReference) bool {
	for _, managedRes := range e.objects {
		if managedRes.Resource == res {
			return true
		}
	}
	return false
}
