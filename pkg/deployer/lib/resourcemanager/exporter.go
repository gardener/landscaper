// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
	"math"
	"sync"
	"time"

	apimacherrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"

	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
)

// ExporterOptions defines the options for the exporter.
type ExporterOptions struct {
	KubeClient     client.Client
	DefaultTimeout *time.Duration

	DeployItem *lsv1alpha1.DeployItem
	LsClient   client.Client

	Objects managedresource.ManagedResourceStatusList
}

// Exporter defines the export of data from manifests.
type Exporter struct {
	kubeClient     client.Client
	defaultTimeout time.Duration

	DeployItem *lsv1alpha1.DeployItem
	LsClient   client.Client

	objects managedresource.ManagedResourceStatusList
}

// NewExporter creates a new exporter.
func NewExporter(opts ExporterOptions) *Exporter {
	exporter := &Exporter{
		kubeClient:     opts.KubeClient,
		defaultTimeout: 5 * time.Minute,
		objects:        opts.Objects,

		DeployItem: opts.DeployItem,
		LsClient:   opts.LsClient,
	}
	if opts.DefaultTimeout != nil {
		exporter.defaultTimeout = *opts.DefaultTimeout
	}

	return exporter
}

// Export exports all keys that are defined in the exports definition.
func (e *Exporter) Export(ctx context.Context, exports *managedresource.Exports) (map[string]interface{}, error) {
	log, _ := logging.FromContextOrNew(ctx, nil)
	log = log.WithName("export")
	ctx = logging.NewContext(ctx, log)
	var allErrs []error

	var (
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
			log2 := log.WithValues(lc.KeyExportKey, export.Key)

			backoff := wait.Backoff{
				Jitter: 1.15,
				Steps:  math.MaxInt32,
				Cap:    export.Timeout.Duration,
			}
			var lastErr error
			if err := wait.ExponentialBackoffWithContext(ctx, backoff, func() (done bool, err error) {

				di := &lsv1alpha1.DeployItem{}
				err = read_write_layer.GetDeployItem(ctx, e.LsClient, client.ObjectKeyFromObject(e.DeployItem), di)
				if err != nil {
					return false, err
				}

				if di.Status.DeployItemPhase == lsv1alpha1.DeployItemPhaseFailed {
					return false, fmt.Errorf("interuppted during export collection")
				}

				value, err := e.doExport(ctx, export)
				if err != nil {
					log2.Debug("error while creating export", lc.KeyError, err.Error())
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

	if export.FromObjectReference != nil {
		var err error
		val, err = e.exportFromReferencedResource(ctx, export, val)
		if err != nil {
			return nil, err
		}
	}

	newValue, err := jsonpath.Construct(export.Key, val)
	if err != nil {
		return nil, err
	}
	return newValue, nil
}

func (e *Exporter) exportFromReferencedResource(ctx context.Context, export managedresource.Export, ref interface{}) (interface{}, error) {
	// check if the ref is of the right type
	refMap, ok := ref.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected reference %#v, expected map with name and namespace", ref)
	}
	refName, ok := refMap["name"]
	if !ok {
		return nil, fmt.Errorf("unexpected reference %#v, expected map to have a name key", ref)
	}
	name, ok := refName.(string)
	if !ok {
		return nil, fmt.Errorf("expected name %#v to be a string", refName)
	}

	namespace := export.FromResource.Namespace // default to same namespace as resource
	refNamespace, ok := refMap["namespace"]
	if ok {
		namespace, ok = refNamespace.(string)
		if !ok {
			return nil, fmt.Errorf("expected namespace %#v to be a string", refName)
		}
	}

	// get resource from client
	obj := kutil.ObjectFromTypedObjectReference(&lsv1alpha1.TypedObjectReference{
		APIVersion: export.FromObjectReference.APIVersion,
		Kind:       export.FromObjectReference.Kind,
		ObjectReference: lsv1alpha1.ObjectReference{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err := e.kubeClient.Get(ctx, kutil.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, err
	}

	var val interface{}
	if err := jsonpath.GetValue(export.FromObjectReference.JSONPath, obj.Object, &val); err != nil {
		return nil, err
	}
	return val, nil
}
