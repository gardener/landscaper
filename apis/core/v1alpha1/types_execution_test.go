// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("Executions", func() {

	Context("Marshal", func() {

		It("marshal and unmarshal an execution spec with compression", func() {
			exec := lsv1alpha1.Execution{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testName",
					Namespace: "testNamespace",
				},
				Spec: lsv1alpha1.ExecutionSpec{
					Context: "testContext",
					DeployItems: []lsv1alpha1.DeployItemTemplate{
						{
							Name: "di1",
							Type: "diType1",
							Target: &lsv1alpha1.ObjectReference{
								Name:      "testTargetName1",
								Namespace: "testTargetNamespace1",
							},
							Labels:             nil,
							Configuration:      nil,
							DependsOn:          nil,
							Timeout:            &lsv1alpha1.Duration{5 * time.Minute},
							UpdateOnChangeOnly: true,
						},
						{
							Name: "di2",
							Type: "diType2",
							Target: &lsv1alpha1.ObjectReference{
								Name:      "testTargetName2",
								Namespace: "testTargetNamespace2",
							},
							DependsOn:          []string{"di1"},
							Timeout:            &lsv1alpha1.Duration{5 * time.Minute},
							UpdateOnChangeOnly: false,
						},
					},
				},
			}

			execBytes, err := json.Marshal(exec)
			Expect(err).NotTo(HaveOccurred())
			Expect(execBytes).NotTo(BeNil())

			exec2 := lsv1alpha1.Execution{}
			err = json.Unmarshal(execBytes, &exec2)
			Expect(err).NotTo(HaveOccurred())
			Expect(exec2).To(Equal(exec))
		})

		It("should marshal an execution without deployitems", func() {
			exec := lsv1alpha1.Execution{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testName",
					Namespace: "testNamespace",
				},
				Spec: lsv1alpha1.ExecutionSpec{
					Context:     "testContext",
					DeployItems: nil,
				},
			}

			execBytes, err := json.Marshal(exec)
			Expect(err).NotTo(HaveOccurred())
			Expect(execBytes).NotTo(BeNil())

			exec2 := lsv1alpha1.Execution{}
			err = json.Unmarshal(execBytes, &exec2)
			Expect(err).NotTo(HaveOccurred())
			Expect(exec2).To(Equal(exec))
		})

		It("should unmarshal an uncompressed execution spec", func() {
			type ExecutionSpecAlt lsv1alpha1.ExecutionSpec

			type ExecutionAlt struct {
				metav1.ObjectMeta `json:"metadata,omitempty"`
				Spec              ExecutionSpecAlt `json:"spec,omitempty"`
			}

			exec := ExecutionAlt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testName",
					Namespace: "testNamespace",
				},
				Spec: ExecutionSpecAlt{
					Context: "testContext",
					DeployItems: []lsv1alpha1.DeployItemTemplate{
						{
							Name: "di1",
							Type: "diType1",
							Target: &lsv1alpha1.ObjectReference{
								Name:      "testTargetName1",
								Namespace: "testTargetNamespace1",
							},
							Labels:             nil,
							Configuration:      nil,
							DependsOn:          nil,
							Timeout:            &lsv1alpha1.Duration{5 * time.Minute},
							UpdateOnChangeOnly: true,
						},
						{
							Name: "di2",
							Type: "diType2",
							Target: &lsv1alpha1.ObjectReference{
								Name:      "testTargetName2",
								Namespace: "testTargetNamespace2",
							},
							DependsOn:          []string{"di1"},
							Timeout:            &lsv1alpha1.Duration{5 * time.Minute},
							UpdateOnChangeOnly: false,
						},
					},
				},
			}

			execBytes, err := json.Marshal(exec)
			Expect(err).NotTo(HaveOccurred())
			Expect(execBytes).NotTo(BeNil())

			exec2 := lsv1alpha1.Execution{}
			err = json.Unmarshal(execBytes, &exec2)
			Expect(err).NotTo(HaveOccurred())
			Expect(exec2.ObjectMeta).To(Equal(exec.ObjectMeta))
			Expect(ExecutionSpecAlt(exec2.Spec)).To(Equal(exec.Spec))
		})
	})

})
