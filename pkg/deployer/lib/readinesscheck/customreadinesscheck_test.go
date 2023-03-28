// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	health "github.com/gardener/landscaper/apis/deployer/utils/readinesschecks"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Custom health checks", func() {

	var (
		customHealthCheck CustomReadinessCheck
	)

	BeforeEach(func() {
		customHealthCheck = CustomReadinessCheck{
			Context:   logging.NewContext(ctx, logging.Discard()),
			Client:    testenv.Client,
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

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: []lsv1alpha1.TypedObjectReference{ref},
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

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: []lsv1alpha1.TypedObjectReference{ref},
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

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: []lsv1alpha1.TypedObjectReference{ref},
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

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: []lsv1alpha1.TypedObjectReference{ref},
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

		obj, err := getObjectsByLabels(ctx, testenv.Client, selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(HaveLen(2))
	})

	It("should select ONE object with the given GVK + namespacedname", func() {
		selector := []lsv1alpha1.TypedObjectReference{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				ObjectReference: lsv1alpha1.ObjectReference{
					Namespace: state.Namespace,
					Name:      "simple-deployment",
				},
			},
		}

		obj, err := getObjectsByTypedReference(ctx, testenv.Client, selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(HaveLen(1))
	})

	It("should select MULTIPLE objects with the given GVK + namespacedname", func() {
		selector := []lsv1alpha1.TypedObjectReference{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				ObjectReference: lsv1alpha1.ObjectReference{
					Namespace: state.Namespace,
					Name:      "simple-deployment",
				},
			},
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				ObjectReference: lsv1alpha1.ObjectReference{
					Namespace: state.Namespace,
					Name:      "deployment-with-labels",
				},
			},
		}

		obj, err := getObjectsByTypedReference(ctx, testenv.Client, selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(HaveLen(2))
	})

	It("should continuously check for requirements being fulfilled", func() {
		testFileName := "04-configmap.yaml"

		testObjects, objectRefs := loadSingleObjectFromFile(testFileName)
		Expect(testObjects).To(HaveLen(1))
		customHealthCheck.ManagedResources = objectRefs
		ref := customHealthCheck.ManagedResources[0]

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Name:     "check " + ref.Kind,
			Resource: []lsv1alpha1.TypedObjectReference{ref},
			Requirements: []health.RequirementSpec{
				{
					JsonPath: ".data.readyOne",
					Operator: selection.Exists,
				},
				{
					JsonPath: "data.readyTwo",
					Operator: selection.Equals,
					Value:    getRawValues("Yes"),
				},
				{
					JsonPath: "data.invalid",
					Operator: selection.DoesNotExist,
				},
			},
		}

		go func() {
			defer GinkgoRecover()
			cm := &corev1.ConfigMap{}
			Expect(state.Client.Get(customHealthCheck.Context,
				types.NamespacedName{
					Name:      customHealthCheck.ManagedResources[0].Name,
					Namespace: state.Namespace}, cm)).To(Succeed())

			time.Sleep(1 * time.Second)
			cm.Data["readyOne"] = "created"
			Expect(state.Client.Update(customHealthCheck.Context, cm)).To(Succeed())

			time.Sleep(1 * time.Second)
			cm.Data["readyTwo"] = "No"
			Expect(state.Client.Update(customHealthCheck.Context, cm)).To(Succeed())

			time.Sleep(1 * time.Second)
			cm.Data["readyTwo"] = "Yes"
			Expect(state.Client.Update(customHealthCheck.Context, cm)).To(Succeed())
		}()

		Eventually(func() bool {
			expectedErr := &ObjectNotReadyError{}
			cm := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			}
			Expect(state.Client.Get(customHealthCheck.Context,
				types.NamespacedName{
					Name:      customHealthCheck.ManagedResources[0].Name,
					Namespace: state.Namespace}, cm)).To(Succeed())
			if err := customHealthCheck.CheckObject(cm); err != nil {
				if errors.As(err, &expectedErr) {
					return false
				} else {
					Fail("error during custom health check")
				}
			}

			Expect(cm.Object["data"]).To(HaveKeyWithValue("readyOne", "created"))
			Expect(cm.Object["data"]).To(HaveKeyWithValue("readyTwo", "Yes"))
			Expect(cm.Object["data"]).ToNot(HaveKey("invalid"))
			return true
		}).WithTimeout(1 * time.Minute).Should(BeTrue())
	})

	It("should perform checks on non-managed resources", func() {
		configmap1 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-managed-1",
				Namespace: state.Namespace,
			},
			Data: map[string]string{
				"ready": "true",
			},
		}

		configmap2 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-managed-2",
				Namespace: state.Namespace,
				Labels: map[string]string{
					"app.name": "non-managed",
				},
			},
			Data: map[string]string{
				"ready": "true",
			},
		}

		configmap1Ref, err := kutil.TypedObjectReferenceFromObject(configmap1, testenv.Client.Scheme())
		Expect(err).ToNot(HaveOccurred())

		configmap2Ref, err := kutil.TypedObjectReferenceFromObject(configmap2, testenv.Client.Scheme())
		Expect(err).ToNot(HaveOccurred())

		customHealthCheck.Configuration = health.CustomReadinessCheckConfiguration{
			Timeout: &lsv1alpha1.Duration{
				Duration: 1 * time.Minute,
			},
			Name: "check-non-managed",
			Resource: []lsv1alpha1.TypedObjectReference{
				*configmap1Ref,
			},
			LabelSelector: &health.LabelSelectorSpec{
				APIVersion: configmap2Ref.APIVersion,
				Kind:       configmap2Ref.Kind,
				Labels: map[string]string{
					"app.name": "non-managed",
				},
			},
			Requirements: []health.RequirementSpec{
				{
					JsonPath: "data.ready",
					Operator: selection.Equals,
					Value:    getRawValues("true"),
				},
			},
		}

		go func() {
			defer GinkgoRecover()
			time.Sleep(1 * time.Second)
			Expect(testenv.Client.Create(ctx, configmap1)).ToNot(HaveOccurred())
			Expect(testenv.Client.Create(ctx, configmap2)).ToNot(HaveOccurred())
		}()

		Eventually(func() bool {
			return customHealthCheck.CheckResourcesReady() == nil
		}).WithTimeout(1 * time.Minute).Should(BeTrue())

	})
})

func loadSingleObjectFromFile(fileName string) ([]*unstructured.Unstructured, []lsv1alpha1.TypedObjectReference) {
	testObjectsRaw, err := os.ReadFile(filepath.Join(testdataDir, fileName))
	Expect(err).ToNot(HaveOccurred())
	testObjects, err := kutil.DecodeObjects(logging.Discard(), fileName, testObjectsRaw)
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
