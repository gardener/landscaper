// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"context"
	"errors"

	"github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
	mock_client "github.com/gardener/landscaper/pkg/utils/mocks/client"
)

var _ = Describe("Reconcile", func() {

	var (
		a                *actuator
		ctrl             *gomock.Controller
		mockClient       *mock_client.MockClient
		mockStatusWriter *mock_client.MockStatusWriter
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mock_client.NewMockClient(ctrl)
		mockStatusWriter = mock_client.NewMockStatusWriter(ctrl)
		mockClient.EXPECT().Status().AnyTimes().Return(mockStatusWriter)
		a = &actuator{
			log:    testing.NullLogger{},
			c:      mockClient,
			scheme: kubernetes.LandscaperScheme,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should merged 2 referenced secret configurations and create a new secret", func() {
		orSecret1 := lsv1alpha1.ObjectReference{Name: "test-sec-1", Namespace: "default"}
		orSecret2 := lsv1alpha1.ObjectReference{Name: "test-sec-2", Namespace: "default"}

		secret1Data := `{ "key1": "value1" }`
		secret2Data := `{ "key2": "value2" }`

		lsConfig := &lsv1alpha1.LandscapeConfiguration{}
		lsConfig.Spec.SecretReferences = []lsv1alpha1.ObjectReference{orSecret1, orSecret2}

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{}, gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Get(gomock.Any(), orSecret1.NamespacedName(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			*secret = corev1.Secret{
				Data: map[string][]byte{
					"data": []byte(secret1Data),
				},
			}
			return nil
		})
		mockClient.EXPECT().Get(gomock.Any(), orSecret2.NamespacedName(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			*secret = corev1.Secret{
				Data: map[string][]byte{
					"data": []byte(secret2Data),
				},
			}
			return nil
		})

		var mergedSecret *corev1.Secret
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, obj *corev1.Secret, opts ...client.CreateOption) error {
			mergedSecret = obj
			return nil
		})

		err := a.Ensure(context.TODO(), lsConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(mergedSecret).ToNot(BeNil())

		var res map[string]interface{}
		err = yaml.Unmarshal(mergedSecret.Data[lsv1alpha1.DataObjectSecretDataKey], &res)
		Expect(err).ToNot(HaveOccurred())
		Expect(lsConfig.Status.Conditions).To(HaveLen(1))
		Expect(lsConfig.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))

		Expect(res["key1"]).To(Equal("value1"))
		Expect(res["key2"]).To(Equal("value2"))
	})

	It("should update the last observed generation of the referenced secrets", func() {
		orSecret1 := lsv1alpha1.ObjectReference{Name: "test-sec-1", Namespace: "default"}
		secret1Data := `{ "key1": "value1" }`

		lsConfig := &lsv1alpha1.LandscapeConfiguration{}
		lsConfig.Spec.SecretReferences = []lsv1alpha1.ObjectReference{orSecret1}

		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{}, gomock.Any()).Times(1).Return(apierrors.NewNotFound(schema.GroupResource{}, ""))
		mockClient.EXPECT().Get(gomock.Any(), orSecret1.NamespacedName(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *corev1.Secret) error {
			*secret = corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 5,
				},
				Data: map[string][]byte{
					"data": []byte(secret1Data),
				},
			}
			return nil
		})

		var mergedSecret *corev1.Secret
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, obj *corev1.Secret, opts ...client.CreateOption) error {
			mergedSecret = obj
			return nil
		})

		err := a.Ensure(context.TODO(), lsConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(mergedSecret).ToNot(BeNil())

		var res map[string]interface{}
		err = yaml.Unmarshal(mergedSecret.Data[lsv1alpha1.DataObjectSecretDataKey], &res)
		Expect(err).ToNot(HaveOccurred())
		Expect(lsConfig.Status.Conditions).To(HaveLen(1))
		Expect(lsConfig.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionTrue))

		Expect(lsConfig.Status.Secrets).To(ConsistOf(lsv1alpha1.VersionedObjectReference{
			ObservedGeneration: 5,
			ObjectReference:    orSecret1,
		}))
	})

	It("should return and report an error in the condition if an error occurs", func() {
		orSecret1 := lsv1alpha1.ObjectReference{Name: "test-sec-1", Namespace: "default"}
		orSecret2 := lsv1alpha1.ObjectReference{Name: "test-sec-2", Namespace: "default"}

		lsConfig := &lsv1alpha1.LandscapeConfiguration{}
		lsConfig.Spec.SecretReferences = []lsv1alpha1.ObjectReference{orSecret1, orSecret2}

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("any error"))
		mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		err := a.Ensure(context.TODO(), lsConfig)
		Expect(err).To(HaveOccurred())

		Expect(lsConfig.Status.Conditions).To(HaveLen(1))
		Expect(lsConfig.Status.Conditions[0].Status).To(Equal(lsv1alpha1.ConditionFalse))
	})

})
