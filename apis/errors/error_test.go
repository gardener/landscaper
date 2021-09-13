// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package errors_test

import (
	"testing"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "errors Test Suite")
}

var _ = Describe("errors", func() {

	Context("GetPhaseForLastError", func() {
		DescribeTable("GetPhaseForLastError",
			func(phase lsv1alpha1.ComponentInstallationPhase, lastError *lsv1alpha1.Error, d time.Duration, expected lsv1alpha1.ComponentInstallationPhase) {
				res := errors.GetPhaseForLastError(phase, lastError, d)
				Expect(res).To(Equal(expected))
			},
			Entry("return the given phase if no error is defined",
				lsv1alpha1.ComponentPhaseInit,
				nil,
				10*time.Second,
				lsv1alpha1.ComponentPhaseInit),
			Entry("return the given phase if an error is defined",
				lsv1alpha1.ComponentPhaseInit,
				errors.NewError("", "", "").LandscaperError(),
				10*time.Second,
				lsv1alpha1.ComponentPhaseInit),
			Entry("return failed phase for unrecoverable errors",
				lsv1alpha1.ComponentPhaseInit,
				errors.NewError("", "", "", lsv1alpha1.ErrorConfigurationProblem).LandscaperError(),
				10*time.Second,
				lsv1alpha1.ComponentPhaseFailed))
	})

})
