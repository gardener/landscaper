package read_write_layer

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("Read Write Layer", func() {

	Context("History", func() {

		It("should init the installation history", func() {
			inst := &lsv1alpha1.Installation{}
			addHistoryItemToInstallationStatus(W000001, inst)
			addHistoryItemToInstallationStatus(W000002, inst)
			addHistoryItemToInstallationStatus(W000003, inst)
			Expect(inst.Status.History).To(HaveLen(3))
		})

		It("should cut the installation history", func() {
			inst := &lsv1alpha1.Installation{}
			for i := 0; i < maxHistoryLenth+5; i++ {
				addHistoryItemToInstallationStatus(W000001, inst)
			}
			addHistoryItemToInstallationStatus(W000002, inst)
			Expect(inst.Status.History).To(HaveLen(maxHistoryLenth))
			Expect(inst.Status.History[maxHistoryLenth-2].WriteID).To(Equal(W000001))
			Expect(inst.Status.History[maxHistoryLenth-1].WriteID).To(Equal(W000002))
		})

		It("should init the execution history", func() {
			exec := &lsv1alpha1.Execution{}
			addHistoryItemToExecutionStatus(W000001, exec)
			addHistoryItemToExecutionStatus(W000002, exec)
			addHistoryItemToExecutionStatus(W000003, exec)
			Expect(exec.Status.History).To(HaveLen(3))
		})

		It("should cut the execution history", func() {
			exec := &lsv1alpha1.Execution{}
			for i := 0; i < maxHistoryLenth+5; i++ {
				addHistoryItemToExecutionStatus(W000001, exec)
			}
			addHistoryItemToExecutionStatus(W000002, exec)
			Expect(exec.Status.History).To(HaveLen(maxHistoryLenth))
			Expect(exec.Status.History[maxHistoryLenth-2].WriteID).To(Equal(W000001))
			Expect(exec.Status.History[maxHistoryLenth-1].WriteID).To(Equal(W000002))
		})

		It("should init the deployitem history", func() {
			di := &lsv1alpha1.DeployItem{}
			addHistoryItemToDeployItemStatus(W000001, di)
			addHistoryItemToDeployItemStatus(W000002, di)
			addHistoryItemToDeployItemStatus(W000003, di)
			Expect(di.Status.History).To(HaveLen(3))
		})

		It("should cut the deployitem history", func() {
			di := &lsv1alpha1.DeployItem{}
			for i := 0; i < maxHistoryLenth+5; i++ {
				addHistoryItemToDeployItemStatus(W000001, di)
			}
			addHistoryItemToDeployItemStatus(W000002, di)
			Expect(di.Status.History).To(HaveLen(maxHistoryLenth))
			Expect(di.Status.History[maxHistoryLenth-2].WriteID).To(Equal(W000001))
			Expect(di.Status.History[maxHistoryLenth-1].WriteID).To(Equal(W000002))
		})
	})
})
