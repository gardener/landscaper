// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package readinesscheck

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	mock_client "github.com/gardener/landscaper/controller-utils/pkg/kubernetes/mock"
)

func createUnstructuredPod() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "core",
		Version: "v1",
		Kind:    "Pod",
	})
	obj.SetName("Foo")
	obj.SetNamespace("default")

	return obj
}

var _ = Describe("IsObjectReady", func() {

	var (
		ctrl       *gomock.Controller
		fakeClient *mock_client.MockClient
		ctx        context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fakeClient = mock_client.NewMockClient(ctrl)
		ctx = context.Background()
	})

	AfterEach(func() {

	})

	It("should return a recoverable error in case of k8s client errors", func() {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "core",
			Version: "v1",
			Kind:    "Pod",
		})
		obj.SetName("Foo")
		obj.SetNamespace("default")

		fakeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("network error"))
		err := IsObjectReady(ctx, fakeClient, obj, func(u *unstructured.Unstructured) error {
			return nil
		})

		Expect(err).To(HaveOccurred())
		Expect(reflect.TypeOf(err)).To(Equal(reflect.TypeOf(&RecoverableError{})))
	})

	It("should return an object not ready error in case the object is not ready", func() {
		obj := createUnstructuredPod()

		fakeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		err := IsObjectReady(ctx, fakeClient, obj, func(u *unstructured.Unstructured) error {
			return NewObjectNotReadyError(u, fmt.Errorf("this is not expected"))
		})

		Expect(err).To(HaveOccurred())
		Expect(reflect.TypeOf(err)).To(Equal(reflect.TypeOf(&ObjectNotReadyError{})))
	})
})

type MockInterruptionChecker struct {
}

func (c *MockInterruptionChecker) Check(_ context.Context) error {
	return nil
}

var _ = Describe("WaitForObjectsReady", func() {
	var (
		ctrl       *gomock.Controller
		fakeClient *mock_client.MockClient
		ctx        context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		fakeClient = mock_client.NewMockClient(ctrl)
		ctx = context.Background()
	})

	AfterEach(func() {

	})

	It("should wait for objects being ready", func() {
		getObjectFuncCalls := 0
		getObjectsFunc := func() ([]*unstructured.Unstructured, error) {
			if getObjectFuncCalls == 0 {
				getObjectFuncCalls += 1
				return nil, NewObjectNotReadyError(createUnstructuredPod(), fmt.Errorf("it is not ready"))
			}
			list := []*unstructured.Unstructured{
				createUnstructuredPod(),
			}
			return list, nil
		}

		checkObjectsFunc := func(u *unstructured.Unstructured) error {
			return nil
		}

		fakeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("network error"))
		fakeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		err := WaitForObjectsReady(ctx, 10*time.Second, fakeClient,
			getObjectsFunc,
			checkObjectsFunc,
			&MockInterruptionChecker{})

		Expect(err).ToNot(HaveOccurred())
	})

})
