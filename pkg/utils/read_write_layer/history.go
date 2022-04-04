package read_write_layer

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	maxHistoryLenth   = 12
	historyAnnotation = "landscaper.gardener.cloud/history"
)

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

func getHistoryAnnotation(meta *metav1.ObjectMeta) ([]lsv1alpha1.HistoryItem, error) {
	rawHistory, ok := meta.Annotations[historyAnnotation]
	if !ok {
		return nil, nil
	}

	history := []lsv1alpha1.HistoryItem{}
	if err := json.Unmarshal([]byte(rawHistory), &history); err != nil {
		return nil, err
	}

	return history, nil
}

func setHistoryAnnotation(meta *metav1.ObjectMeta, history []lsv1alpha1.HistoryItem) error {
	rawHistory, err := json.Marshal(history)
	if err != nil {
		return err
	}

	if len(meta.Annotations) == 0 {
		meta.Annotations = map[string]string{}
	}

	meta.Annotations[historyAnnotation] = string(rawHistory)
	return nil
}

func addHistoryItemToInstallation(writeID WriteID, installation *lsv1alpha1.Installation) error {
	history, err := getHistoryAnnotation(&installation.ObjectMeta)
	if err != nil {
		return err
	}

	item := newHistoryItem(writeID, string(installation.Status.Phase))
	history = cutHistory(append(history, *item))
	return setHistoryAnnotation(&installation.ObjectMeta, history)
}

func addHistoryItemToInstallationStatus(writeID WriteID, installation *lsv1alpha1.Installation) {
	item := newHistoryItem(writeID, string(installation.Status.Phase))
	installation.Status.History = cutHistory(append(installation.Status.History, *item))
}

func addHistoryItemToExecution(writeID WriteID, execution *lsv1alpha1.Execution) error {
	history, err := getHistoryAnnotation(&execution.ObjectMeta)
	if err != nil {
		return err
	}

	item := newHistoryItem(writeID, string(execution.Status.Phase))
	history = cutHistory(append(history, *item))
	return setHistoryAnnotation(&execution.ObjectMeta, history)
}

func addHistoryItemToExecutionStatus(writeID WriteID, execution *lsv1alpha1.Execution) {
	item := newHistoryItem(writeID, string(execution.Status.Phase))
	execution.Status.History = cutHistory(append(execution.Status.History, *item))
}

func addHistoryItemToDeployItem(writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	history, err := getHistoryAnnotation(&deployItem.ObjectMeta)
	if err != nil {
		return err
	}

	item := newHistoryItem(writeID, string(deployItem.Status.Phase))
	history = cutHistory(append(history, *item))
	return setHistoryAnnotation(&deployItem.ObjectMeta, history)
}

func addHistoryItemToDeployItemStatus(writeID WriteID, deployItem *lsv1alpha1.DeployItem) {
	item := newHistoryItem(writeID, string(deployItem.Status.Phase))
	deployItem.Status.History = cutHistory(append(deployItem.Status.History, *item))
}
