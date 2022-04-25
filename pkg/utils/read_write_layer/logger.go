package read_write_layer

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

const (
	historyLogLevel       = 1
	keyName               = "name"
	keyNamespace          = "namespace"
	keyPhase              = "phase"
	keyWriteID            = "write-id"
	keyGenerationOld      = "generation-old"
	keyGenerationNew      = "generation-new"
	keyResourceVersionOld = "resource-version-old"
	keyResourceVersionNew = "resource-version-new"
)

func (w *Writer) logTargetUpdate(writeID WriteID, op string, target *lsv1alpha1.Target,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(target)
		w.log.V(historyLogLevel).Info(op,
			keyWriteID, writeID,
			keyName, target.Name,
			keyNamespace, target.Namespace,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, op,
			keyWriteID, writeID,
			keyName, target.Name,
			keyNamespace, target.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDataObjectUpdate(writeID WriteID, op string, do *lsv1alpha1.DataObject,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(do)
		w.log.V(historyLogLevel).Info(op,
			keyWriteID, writeID,
			keyName, do.Name,
			keyNamespace, do.Namespace,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, op,
			keyWriteID, writeID,
			keyName, do.Name,
			keyNamespace, do.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logInstallationUpdate(writeID WriteID, op string, installation *lsv1alpha1.Installation,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(installation)
		w.log.V(historyLogLevel).Info(op,
			keyWriteID, writeID,
			keyName, installation.Name,
			keyNamespace, installation.Namespace,
			keyPhase, installation.Status.Phase,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, op,
			keyWriteID, writeID,
			keyName, installation.Name,
			keyNamespace, installation.Namespace,
			keyPhase, installation.Status.Phase,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logExecutionUpdate(writeID WriteID, op string, execution *lsv1alpha1.Execution,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(execution)
		w.log.V(historyLogLevel).Info(op,
			keyWriteID, writeID,
			keyName, execution.Name,
			keyNamespace, execution.Namespace,
			keyPhase, execution.Status.Phase,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, op,
			keyWriteID, writeID,
			keyName, execution.Name,
			keyNamespace, execution.Namespace,
			keyPhase, execution.Status.Phase,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDeployItemUpdate(writeID WriteID, op string, deployItem *lsv1alpha1.DeployItem,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(deployItem)
		w.log.V(historyLogLevel).Info(op,
			keyWriteID, writeID,
			keyName, deployItem.Name,
			keyNamespace, deployItem.Namespace,
			keyPhase, deployItem.Status.Phase,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, op,
			keyWriteID, writeID,
			keyName, deployItem.Name,
			keyNamespace, deployItem.Namespace,
			keyPhase, deployItem.Status.Phase,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}
