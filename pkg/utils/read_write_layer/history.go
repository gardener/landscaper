package read_write_layer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const maxHistoryLenth = 12

func newHistoryItem(writeID WriteID, phase string) *lsv1alpha1.HistoryItem {
	return &lsv1alpha1.HistoryItem{
		UpdateTime: metav1.Now(),
		WriteID:    writeID,
		Phase:      phase,
	}
}

func cutHistory(history []lsv1alpha1.HistoryItem) []lsv1alpha1.HistoryItem {
	length := len(history)
	if length <= maxHistoryLenth {
		return history
	}
	return history[length-maxHistoryLenth : length]
}

func addHistoryItemToInstallationStatus(writeID WriteID, installation *lsv1alpha1.Installation) {
	item := newHistoryItem(writeID, string(installation.Status.Phase))
	installation.Status.History = cutHistory(append(installation.Status.History, *item))
}

func addHistoryItemToExecutionStatus(writeID WriteID, execution *lsv1alpha1.Execution) {
	item := newHistoryItem(writeID, string(execution.Status.Phase))
	execution.Status.History = cutHistory(append(execution.Status.History, *item))
}

func addHistoryItemToDeployItemStatus(writeID WriteID, deployItem *lsv1alpha1.DeployItem) {
	item := newHistoryItem(writeID, string(deployItem.Status.Phase))
	deployItem.Status.History = cutHistory(append(deployItem.Status.History, *item))
}
