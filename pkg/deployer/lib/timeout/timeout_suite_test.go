package timeout

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timeout Test Suite")
}

var _ = Describe("Timeout checker", func() {

	Context("activation", func() {
		AfterEach(func() {
			ActivateStandardTimeoutChecker()
		})

		It("should set the timeout checker instance", func() {
			ActivateIgnoreTimeoutChecker()
			Expect(timeoutCheckerInstance).NotTo(BeNil())
			_, ok := timeoutCheckerInstance.(*ignoreTimeoutChecker)
			Expect(ok).To(BeTrue())

			ActivateCheckpointTimeoutChecker("test")
			Expect(timeoutCheckerInstance).NotTo(BeNil())
			_, ok = timeoutCheckerInstance.(*checkpointTimeoutChecker)
			Expect(ok).To(BeTrue())

			ActivateStandardTimeoutChecker()
			Expect(timeoutCheckerInstance).NotTo(BeNil())
			_, ok = timeoutCheckerInstance.(*standardTimeoutChecker)
			Expect(ok).To(BeTrue())
		})
	})

	Context("standard implementation", func() {

		buildDeployItemWithTimeoutData := func(initTime time.Time, timeoutDuration time.Duration) *lsv1alpha1.DeployItem {
			return &lsv1alpha1.DeployItem{
				Spec: lsv1alpha1.DeployItemSpec{
					Timeout: &lsv1alpha1.Duration{Duration: timeoutDuration},
				},
				Status: lsv1alpha1.DeployItemStatus{
					TransitionTimes: &lsv1alpha1.TransitionTimes{
						InitTime: &metav1.Time{
							Time: initTime,
						},
					},
				},
			}
		}

		It("should accept a deploy item whose timeout is not exceeded", func() {
			deployItem := buildDeployItemWithTimeoutData(time.Now(), time.Minute)
			remainingDuration, err := TimeoutExceeded(context.Background(), deployItem, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(remainingDuration >= 0).To(BeTrue())
			Expect(remainingDuration <= time.Minute).To(BeTrue())
		})

		It("should detect an exceeded timeout", func() {
			deployItem := buildDeployItemWithTimeoutData(time.Now().Add(-2*time.Minute), time.Minute)
			_, err := TimeoutExceeded(context.Background(), deployItem, "")
			Expect(err).To(HaveOccurred())
			Expect(err.LandscaperError()).NotTo(BeNil())
			Expect(err.LandscaperError().Codes).To(ContainElement(lsv1alpha1.ErrorTimeout))
		})
	})
})
