// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"
	lserror "github.com/gardener/landscaper/apis/errors"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/utils"
)

// CustomReadinessCheck contains all the data and methods required to kick off a custom readiness check
type CustomReadinessCheck struct {
	Context          context.Context
	Client           client.Client
	CurrentOp        string
	Timeout          *lsv1alpha1.Duration
	ManagedResources []lsv1alpha1.TypedObjectReference
	Configuration    health.CustomReadinessCheckConfiguration
}

// CheckResourcesReady starts a custom readiness check by checking the readiness of the submitted resources
func (c *CustomReadinessCheck) CheckResourcesReady() error {
	if c.Configuration.Disabled || len(c.ManagedResources) == 0 {
		// nothing to do
		return nil
	}

	var objects []*unstructured.Unstructured

	if c.Configuration.Resource != nil {
		objects = getObjectsByTypedReference(c.ManagedResources, c.Configuration.Resource)
	}

	if c.Configuration.LabelSelector != nil {
		o, err := getObjectsByLabels(c.Context, c.Client, c.ManagedResources, c.Configuration.LabelSelector)
		if err != nil {
			return lserror.NewWrappedError(err, c.CurrentOp, "get objects by LabelSelector", err.Error(), lsv1alpha1.ErrorInternalProblem)
		}
		objects = append(objects, o...)
	}

	timeout := c.Timeout.Duration
	if err := WaitForObjectsReady(c.Context, timeout, c.Client, objects, c.CheckObject); err != nil {
		return lserror.NewWrappedError(err,
			c.CurrentOp, "CheckResourceReadiness", err.Error(), lsv1alpha1.ErrorReadinessCheckTimeout)
	}

	return nil
}

// CheckObject checks the readiness of an object and returns an error if the object is considered unready
func (c *CustomReadinessCheck) CheckObject(u *unstructured.Unstructured) error {
	for _, requirement := range c.Configuration.Requirements {
		fields, err := getFieldsByJSONPath(u.Object, requirement.JsonPath)
		if err != nil {
			return lserror.NewWrappedError(err, c.CurrentOp, "parsing JSON path", err.Error())
		}

		if fieldDoesNotExist(fields) {
			if requirement.Operator == selection.DoesNotExist {
				continue
			}
			return NewObjectNotReadyError(u, lserror.NewError(c.CurrentOp, "object check", fmt.Sprintf("field with JSON path %s does not exist", requirement.JsonPath)))
		}

		if requirement.Operator == selection.Exists {
			// field exists and that is all we need to know for selection.Exists
			continue
		}

		requirementValues, err := parseRequirementValues(requirement.Value)
		if err != nil {
			return lserror.NewWrappedError(err, c.CurrentOp, "parse requirement values", err.Error())
		}

		for _, field := range fields {
			for _, value := range field {
				ok, err := matchResourceConditions(value.Interface(), requirementValues, requirement.Operator)
				if err != nil {
					return lserror.NewWrappedError(err, c.CurrentOp, "check resource requirements", err.Error())
				}

				if !ok {
					return NewObjectNotReadyError(u, lserror.NewError(c.CurrentOp, "check object values",
						fmt.Sprintf("resource requirement is not fulfiled for field %s", requirement.JsonPath)))
				}
			}
		}
	}
	return nil
}

func matchResourceConditions(object interface{}, values []interface{}, operator selection.Operator) (bool, error) {
	success := false

	// this is necessary to have numbers(even int) represented as float64 in both objects to compare
	o, err := utils.JSONSerializeToGenericObject(object)
	if err != nil {
		return false, err
	}

	switch operator {

	case selection.Equals, selection.DoubleEquals, selection.In:
		// for the =, == and != cases, it is already guaranteed by the validator that we only have one value to compare against so we can safely do this...
		for _, v := range values {
			if reflect.DeepEqual(o, v) {
				return true, nil
			}
		}
		return false, nil

	case selection.NotEquals:
		// ... or this
		return !reflect.DeepEqual(o, values[0]), nil

	case selection.NotIn:
		found := false
		for _, v := range values {
			found = found || reflect.DeepEqual(o, v)
		}
		return !found, nil
	}

	return success, nil
}

// getObjectsByTypedReference returns an object from a list of TypedObjectReferences identified by a given TypedObjectReference as unstructured.Unstructured
func getObjectsByTypedReference(objects []lsv1alpha1.TypedObjectReference, key []lsv1alpha1.TypedObjectReference) []*unstructured.Unstructured {
	var results []*unstructured.Unstructured

	for _, o := range objects {
		for _, k := range key {
			if o == k {
				obj := kutil.ObjectFromTypedObjectReference(&o)
				results = append(results, obj)
			}
		}
	}

	return results
}

// getObjectsByLabels returns all objects from a list of TypedObjectReferences that match a certain label selector as a slice of unstructured.Unstructured
func getObjectsByLabels(ctx context.Context, client client.Client, objects []lsv1alpha1.TypedObjectReference, selector *health.LabelSelectorSpec) ([]*unstructured.Unstructured, error) {
	var results []*unstructured.Unstructured

	selectorGv, err := schema.ParseGroupVersion(selector.APIVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse group/version of selector %s", selector.APIVersion)
	}
	selectorGvk := selectorGv.WithKind(selector.Kind)

	for _, o := range objects {
		obj := kutil.ObjectFromTypedObjectReference(&o)
		if obj.GroupVersionKind() != selectorGvk {
			continue
		}

		key := kutil.ObjectKey(obj.GetName(), obj.GetNamespace())
		if err := client.Get(ctx, key, obj); err != nil {
			return nil, errors.Wrapf(err, "unable to get object %s %s/%s", obj.GroupVersionKind().String(), obj.GetName(), obj.GetNamespace())
		}

		objectLabels := labels.Set(obj.GetLabels())

		found := true
		for key, value := range selector.Labels {
			if !objectLabels.Has(key) {
				found = false
				continue
			}
			if objectLabels.Get(key) != value {
				found = false
			}
		}

		if found {
			results = append(results, obj)
		}
	}

	return results, nil
}

// getFieldsByJSONPath returns a field from an object identified by its JSON path
func getFieldsByJSONPath(obj map[string]interface{}, fieldPath string) ([][]reflect.Value, error) {
	if !strings.HasPrefix(fieldPath, ".") {
		fieldPath = "." + fieldPath
	}

	p := jsonpath.New("fieldPath").AllowMissingKeys(true)
	if err := p.Parse(fmt.Sprintf("{%s}", fieldPath)); err != nil {
		return nil, errors.Wrapf(err, "cannot parse fieldPath %s", fieldPath)
	}

	results, err := p.FindResults(obj)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find results")
	}

	return results, nil
}

func fieldDoesNotExist(f [][]reflect.Value) bool {
	if len(f) < 1 {
		return true
	}
	if len(f[0]) < 1 {
		return true
	}
	return false
}

func parseRequirementValues(values []runtime.RawExtension) ([]interface{}, error) {
	parsedValues := []interface{}{}
	for i, successValue := range values {
		var tmp map[string]interface{}
		if err := json.Unmarshal(successValue.Raw, &tmp); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal object at index %d", i)
		}

		if val, ok := tmp["value"]; ok {
			parsedValues = append(parsedValues, val)
		} else {
			return nil, errors.Errorf("object at index %d does not contain the value key", i)
		}
	}
	return parsedValues, nil
}
