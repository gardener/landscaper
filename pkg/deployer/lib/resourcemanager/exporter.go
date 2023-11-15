// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resourcemanager

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/utils/managedresource"
	"github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/deployer/lib/interruption"
	"github.com/gardener/landscaper/pkg/deployer/lib/timeout"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/utils"
)

// ExporterOptions defines the options for the exporter.
type ExporterOptions struct {
	KubeClient          client.Client
	InterruptionChecker interruption.InterruptionChecker
	DeployItem          *lsv1alpha1.DeployItem
}

// Exporter defines the export of data from manifests.
type Exporter struct {
	kubeClient          client.Client
	interruptionChecker interruption.InterruptionChecker
	deployItem          *lsv1alpha1.DeployItem
}

// NewExporter creates a new exporter.
func NewExporter(opts ExporterOptions) *Exporter {
	exporter := &Exporter{
		kubeClient:          opts.KubeClient,
		interruptionChecker: opts.InterruptionChecker,
		deployItem:          opts.DeployItem,
	}

	if exporter.interruptionChecker == nil {
		exporter.interruptionChecker = interruption.NewIgnoreInterruptionChecker()
	}

	return exporter
}

func (e *Exporter) Export(ctx context.Context, exports *managedresource.Exports) (map[string]interface{}, error) {
	log, _ := logging.FromContextOrNew(ctx, nil)
	log = log.WithName("export")
	ctx = logging.NewContext(ctx, log)
	var result map[string]interface{}

	for _, export := range exports.Exports {
		if export.FromResource == nil {
			// ignore exports without from resource
			// this currently only used for helm values where no resource is needed.
			continue
		}

		checkpoint := fmt.Sprintf("deployer: during export - key: %s", export.Key)
		timeout, timeoutErr := timeout.TimeoutExceeded(ctx, e.deployItem, checkpoint)
		if timeoutErr != nil {
			return nil, timeoutErr
		}

		log2 := log.WithValues(lc.KeyExportKey, export.Key)

		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
			if err := e.interruptionChecker.Check(ctx); err != nil {
				return false, err
			}

			value, err := e.doExport(ctx, export)
			if err != nil {
				log2.Info("error while creating export", lc.KeyError, err.Error())
				return false, nil
			}
			result = utils.MergeMaps(result, value)
			return true, nil
		})

		if wait.Interrupted(err) {
			msg := fmt.Sprintf("timeout at: %q", checkpoint)
			return nil, errors.NewWrappedError(err, "Export", lsv1alpha1.ProgressingTimeoutReason, msg, lsv1alpha1.ErrorTimeout)
		}

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (e *Exporter) doExport(ctx context.Context, export managedresource.Export) (map[string]interface{}, error) {
	// get resource from client
	obj := kutil.ObjectFromTypedObjectReference(export.FromResource)
	if err := read_write_layer.GetUnstructured(ctx, e.kubeClient, kutil.ObjectKeyFromObject(obj), obj,
		read_write_layer.R000046); err != nil {
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

	if err := read_write_layer.GetUnstructured(ctx, e.kubeClient, kutil.ObjectKeyFromObject(obj), obj,
		read_write_layer.R000047); err != nil {
		return nil, err
	}

	var val interface{}
	if err := jsonpath.GetValue(export.FromObjectReference.JSONPath, obj.Object, &val); err != nil {
		return nil, err
	}
	return val, nil
}
