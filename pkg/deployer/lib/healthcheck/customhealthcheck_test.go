// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthcheck

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	health "github.com/gardener/landscaper/apis/deployer/utils/healthchecks"

	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
)

var _ = Describe("Custom health checks", func() {

	var (
		customHealthCheck CustomHealthCheck
	)

	BeforeEach(func() {
		customHealthCheck = CustomHealthCheck{
			Context:   ctx,
			Client:    testenv.Client,
			Log:       logr.Discard(),
			CurrentOp: "custom health check test",
			Timeout:   &lsv1alpha1.Duration{Duration: 180 * time.Second},
		}
	})

	It("should perform a health check on a Deployment with a custom jsonpath and simple value", func() {
		testFileName := "01-simple-deployment.yaml"
		successValue := int32(2)

		testObjects, objectRefs := loadSingleObjectFromFile(testFileName)
		Expect(testObjects).To(HaveLen(1))
		customHealthCheck.ManagedResources = objectRefs
		ref := customHealthCheck.ManagedResources[0]

		customHealthCheck.Configuration = health.CustomHealthCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: &ref,
			Requirements: []health.RequirementSpec{
				{
					JsonPath: ".status.readyReplicas",
					Operator: selection.Equals,
					Value:    getRawValues(successValue),
				},
			},
		}

		err := customHealthCheck.CheckObject(testObjects[0])
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail on a Deployment with a non-matching replica count", func() {
		testFileName := "01-simple-deployment.yaml"
		failValue := int32(1)

		testObjects, objectRefs := loadSingleObjectFromFile(testFileName)
		Expect(testObjects).To(HaveLen(1))
		customHealthCheck.ManagedResources = objectRefs
		ref := customHealthCheck.ManagedResources[0]

		customHealthCheck.Configuration = health.CustomHealthCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: &ref,
			Requirements: []health.RequirementSpec{
				{
					JsonPath: ".status.readyReplicas",
					Operator: selection.Equals,
					Value:    getRawValues(failValue),
				},
			},
		}

		err := customHealthCheck.CheckObject(testObjects[0])
		Expect(err).To(HaveOccurred())
	})

	It("should successfully perform a health check on a resource with a nested compare value", func() {
		testFileName := "03-deployment-with-nested-status.yaml"
		successValue := map[string]interface{}{
			"slicedStuff": []map[string]interface{}{
				{
					"name":  "foo",
					"value": "bar",
				},
				{
					"name":  "john",
					"value": "doe",
				},
			},
			"nestedStuff": map[string]interface{}{
				"name": map[string]interface{}{
					"first": map[string]interface{}{
						"value": "john",
					},
					"last": map[string]interface{}{
						"value": "doe",
					},
				},
			},
		}

		testObjects, objectRefs := loadSingleObjectFromFile(testFileName)
		Expect(testObjects).To(HaveLen(1))
		customHealthCheck.ManagedResources = objectRefs
		ref := customHealthCheck.ManagedResources[0]

		customHealthCheck.Configuration = health.CustomHealthCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: &ref,
			Requirements: []health.RequirementSpec{
				{
					JsonPath: ".status.additionalTestFields",
					Operator: selection.Equals,
					Value:    getRawValues(successValue),
				},
			},
		}

		err := customHealthCheck.CheckObject(testObjects[0])
		Expect(err).ToNot(HaveOccurred())
	})

	It("should succed on a test that checks for a missing field", func() {
		testFileName := "01-simple-deployment.yaml"

		testObjects, objectRefs := loadSingleObjectFromFile(testFileName)
		Expect(testObjects).To(HaveLen(1))
		customHealthCheck.ManagedResources = objectRefs
		ref := customHealthCheck.ManagedResources[0]

		customHealthCheck.Configuration = health.CustomHealthCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: &ref,
			Requirements: []health.RequirementSpec{
				{
					JsonPath: ".status.fieldNameThatWillNotExist",
					Operator: selection.DoesNotExist,
				},
			},
		}

		err := customHealthCheck.CheckObject(testObjects[0])
		Expect(err).ToNot(HaveOccurred())
	})

	It("should properly select objects with matching labels", func() {
		selector := &health.LabelSelectorSpec{
			APIVersion: "apps/v1",
			Kind:       "Deployment",

			Labels: map[string]string{
				"healthcheck": "testme",
			},
		}

		obj, err := getObjectsByLabels(ctx, testenv.Client, objectRefs, selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(HaveLen(2))
	})

	It("should select the object with the given GVK + namespacedname", func() {
		selector := &lsv1alpha1.TypedObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			ObjectReference: lsv1alpha1.ObjectReference{
				Namespace: state.Namespace,
				Name:      "simple-deployment",
			},
		}

		obj := getObjectsByTypedReference(objectRefs, *selector)
		Expect(obj).To(HaveLen(1))
	})

})

func loadSingleObjectFromFile(fileName string) ([]*unstructured.Unstructured, []lsv1alpha1.TypedObjectReference) {
	testObjectsRaw, err := ioutil.ReadFile(filepath.Join(testdataDir, fileName))
	Expect(err).ToNot(HaveOccurred())
	testObjects, err := kutil.DecodeObjects(logr.Discard(), fileName, testObjectsRaw)
	Expect(err).ToNot(HaveOccurred())

	objectRefs := make([]lsv1alpha1.TypedObjectReference, len(testObjects))
	for i, o := range testObjects {
		objectRefs[i] = *kutil.TypedObjectReferenceFromUnstructuredObject(o)
	}
	return testObjects, objectRefs
}

func getRawValues(v interface{}) []runtime.RawExtension {
	rawValues, err := json.Marshal(map[string]interface{}{
		"value": v,
	})

	if err != nil {
		return nil
	}

	return []runtime.RawExtension{
		{
			Raw: rawValues,
		},
	}
}
