// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

var _ = Describe("DeployItem Classification", func() {

	buildExecutionItem := func(name string, dependsOn []string, jobID, jobIDFinished string, phase lsv1alpha1.DeployItemPhase) *executionItem {
		return &executionItem{
			Info: lsv1alpha1.DeployItemTemplate{
				Name:      name,
				DependsOn: dependsOn,
			},
			DeployItem: &lsv1alpha1.DeployItem{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ""},
				Status: lsv1alpha1.DeployItemStatus{
					JobID:         jobID,
					JobIDFinished: jobIDFinished,
					Phase:         phase,
				},
			},
		}
	}

	buildExecutionItemWithoutDeployItem := func(name string, dependsOn []string) *executionItem {
		return &executionItem{
			Info: lsv1alpha1.DeployItemTemplate{
				Name:      name,
				DependsOn: dependsOn,
			},
			DeployItem: nil,
		}
	}

	It("should treat a missing DeployItem as failed", func() {
		currJobID := "02"
		items := []*executionItem{
			buildExecutionItemWithoutDeployItem("a", nil),
		}

		classification, err := newDeployItemClassification(currJobID, items)
		Expect(err).NotTo(HaveOccurred())

		Expect(classification.succeededItems).To(BeEmpty())
		Expect(classification.failedItems).To(ConsistOf(items[0]))
		Expect(classification.runningItems).To(BeEmpty())
		Expect(classification.runnableItems).To(BeEmpty())
		Expect(classification.pendingItems).To(BeEmpty())
	})

	It("should classify execution items", func() {
		currJobID := "02"
		prevJobID := "01"
		items := []*executionItem{
			buildExecutionItem("a", []string{}, currJobID, currJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("b", []string{"a"}, currJobID, currJobID, lsv1alpha1.DeployItemPhases.Failed),
			buildExecutionItem("c", []string{"a"}, currJobID, currJobID, lsv1alpha1.DeployItemPhases.Succeeded),

			buildExecutionItem("d", []string{}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("e", []string{"a", "c"}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("f", []string{"a", "d"}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("g", []string{"f"}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Failed),

			buildExecutionItem("h", []string{}, currJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("i", []string{}, currJobID, prevJobID, lsv1alpha1.DeployItemPhases.Progressing),
		}

		classification, err := newDeployItemClassification(currJobID, items)
		Expect(err).NotTo(HaveOccurred())

		Expect(classification.succeededItems).To(ConsistOf(items[0], items[2]))
		Expect(classification.failedItems).To(ConsistOf(items[1]))
		Expect(classification.runningItems).To(ConsistOf(items[7], items[8]))
		Expect(classification.runnableItems).To(ConsistOf(items[3], items[4]))
		Expect(classification.pendingItems).To(ConsistOf(items[5], items[6]))
	})

	It("should classify execution items for delete", func() {
		currJobID := "02"
		prevJobID := "01"
		items := []*executionItem{
			buildExecutionItemWithoutDeployItem("a", []string{"c", "f"}),
			buildExecutionItem("b", []string{}, currJobID, currJobID, lsv1alpha1.DeployItemPhases.Failed),
			buildExecutionItemWithoutDeployItem("c", []string{"e"}),

			buildExecutionItem("d", []string{"f"}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("e", []string{}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("f", []string{"g"}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("g", []string{}, prevJobID, prevJobID, lsv1alpha1.DeployItemPhases.Failed),

			buildExecutionItem("h", []string{}, currJobID, prevJobID, lsv1alpha1.DeployItemPhases.Succeeded),
			buildExecutionItem("i", []string{}, currJobID, prevJobID, lsv1alpha1.DeployItemPhases.Deleting),

			buildExecutionItem("j", []string{}, currJobID, currJobID, lsv1alpha1.DeployItemPhases.Succeeded),
		}

		classification, err := newDeployItemClassificationForDelete(currJobID, items)
		Expect(err).NotTo(HaveOccurred())

		Expect(classification.succeededItems).To(ConsistOf(items[0], items[2]))
		Expect(classification.failedItems).To(ConsistOf(items[1]))
		Expect(classification.runningItems).To(ConsistOf(items[7], items[8], items[9]))
		Expect(classification.runnableItems).To(ConsistOf(items[3], items[4]))
		Expect(classification.pendingItems).To(ConsistOf(items[5], items[6]))
	})
})
