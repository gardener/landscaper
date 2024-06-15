// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprints_test

import (
	"os"
	"testing"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/blueprints"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Blueprints Test Suite")
}

var _ = Describe("Internal Blueprint", func() {

	Context("Subinstallations", func() {
		It("should fail if the defined file path does not exist ", func() {
			b := blueprints.New(&lsv1alpha1.Blueprint{}, memoryfs.New())
			b.Info.Subinstallations = []lsv1alpha1.SubinstallationTemplate{
				{
					File: "mypath",
				},
			}

			_, err := b.GetSubinstallations()
			Expect(err).To(HaveOccurred())
			allErrs, ok := err.(utilerrors.Aggregate)
			Expect(ok).To(BeTrue())
			Expect(allErrs.Errors()).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotFound),
				"Field": Equal("subinstallations[0].file"),
			}))))
		})

		It("should fail if a InstallationTemplate defined by a file is invalid", func() {
			b := blueprints.New(&lsv1alpha1.Blueprint{}, memoryfs.New())
			b.Info.Subinstallations = []lsv1alpha1.SubinstallationTemplate{
				{
					File: "mypath",
				},
			}

			installationTemplateBytes := []byte(`wrong type`)
			Expect(vfs.WriteFile(b.Fs, "mypath", installationTemplateBytes, os.ModePerm)).To(Succeed())

			_, err := b.GetSubinstallations()
			Expect(err).To(HaveOccurred())
			allErrs, ok := err.(utilerrors.Aggregate)
			Expect(ok).To(BeTrue())
			Expect(allErrs).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("subinstallations[0].file"),
			}))))
		})
	})

})
