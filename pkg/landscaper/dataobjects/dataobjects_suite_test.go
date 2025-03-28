// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dataobjects

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

func TestTestDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DataObjects Suite")
}

var _ = Describe("Targets", func() {

	It("should compute hashable content for a target with secretRef", func() {
		t := &lsv1alpha1.Target{
			Spec: lsv1alpha1.TargetSpec{
				SecretRef: &lsv1alpha1.LocalSecretReference{
					Name: "testname01",
					Key:  "testkey01",
				},
			},
		}
		content := GetHashableContent(t)

		o := &objectWithSecretRef{}
		err := json.Unmarshal(content, &o)
		Expect(err).NotTo(HaveOccurred())
		Expect(o.SecretRef.Name).To(Equal(t.Spec.SecretRef.Name))
		Expect(o.SecretRef.Key).To(Equal(t.Spec.SecretRef.Key))
	})

})
