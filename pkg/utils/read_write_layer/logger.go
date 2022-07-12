package read_write_layer

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/utils"
)

const (
	historyLogLevel       = 1
	keyName               = "name"
	keyNamespace          = "namespace"
	keyPhase              = "phase"
	keyPhaseDeployer      = "phase-d"
	keyWriteID            = "write-id"
	keyGenerationOld      = "gen-old"
	keyGenerationNew      = "gen-new"
	keyOperation          = "op"
	keyResourceVersionOld = "rv-old"
	keyResourceVersionNew = "rv-new"
	keyJobID              = "j-id"
	keyJobIDFinished      = "j-id-f"
)

func (w *Writer) logContextUpdate(writeID WriteID, msg string, context *lsv1alpha1.Context,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(context)
		w.log.V(historyLogLevel).Info(msg,
			keyWriteID, writeID,
			keyName, context.Name,
			keyNamespace, context.Namespace,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, msg,
			keyWriteID, writeID,
			keyName, context.Name,
			keyNamespace, context.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logTargetUpdate(writeID WriteID, msg string, target *lsv1alpha1.Target,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(target)
		w.log.V(historyLogLevel).Info(msg,
			keyWriteID, writeID,
			keyName, target.Name,
			keyNamespace, target.Namespace,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, msg,
			keyWriteID, writeID,
			keyName, target.Name,
			keyNamespace, target.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDataObjectUpdate(writeID WriteID, msg string, do *lsv1alpha1.DataObject,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(do)
		w.log.V(historyLogLevel).Info(msg,
			keyWriteID, writeID,
			keyName, do.Name,
			keyNamespace, do.Namespace,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.log.Error(err, msg,
			keyWriteID, writeID,
			keyName, do.Name,
			keyNamespace, do.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logInstallationUpdate(writeID WriteID, msg string, installation *lsv1alpha1.Installation,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(installation)
		opNew := lsv1alpha1helper.GetOperation(installation.ObjectMeta)
		if utils.IsNewReconcile() {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
				keyPhase, installation.Status.InstallationPhase,
				keyJobID, installation.Status.JobID,
				keyJobIDFinished, installation.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		} else {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
				keyPhase, installation.Status.Phase,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		}

	} else {
		if utils.IsNewReconcile() {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
				keyPhase, installation.Status.InstallationPhase,
				keyJobID, installation.Status.JobID,
				keyJobIDFinished, installation.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
				keyPhase, installation.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}

func (w *Writer) logExecutionUpdate(writeID WriteID, msg string, execution *lsv1alpha1.Execution,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(execution)
		opNew := lsv1alpha1helper.GetOperation(execution.ObjectMeta)
		if utils.IsNewReconcile() {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
				keyPhase, execution.Status.ExecutionPhase,
				keyJobID, execution.Status.JobID,
				keyJobIDFinished, execution.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		} else {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
				keyPhase, execution.Status.Phase,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		}
	} else {
		if utils.IsNewReconcile() {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
				keyPhase, execution.Status.ExecutionPhase,
				keyJobID, execution.Status.JobID,
				keyJobIDFinished, execution.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
				keyPhase, execution.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}

func (w *Writer) logDeployItemUpdate(writeID WriteID, msg string, deployItem *lsv1alpha1.DeployItem,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(deployItem)
		opNew := lsv1alpha1helper.GetOperation(deployItem.ObjectMeta)
		if utils.IsNewReconcile() {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
				keyPhase, deployItem.Status.DeployItemPhase,
				keyPhaseDeployer, deployItem.Status.Phase,
				keyJobID, deployItem.Status.JobID,
				keyJobIDFinished, deployItem.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		} else {
			w.log.V(historyLogLevel).Info(msg,
				keyWriteID, writeID,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
				keyPhase, deployItem.Status.Phase,
				keyGenerationOld, generationOld,
				keyGenerationNew, generationNew,
				keyOperation, opNew,
				keyResourceVersionOld, resourceVersionOld,
				keyResourceVersionNew, resourceVersionNew,
			)
		}

	} else {
		if utils.IsNewReconcile() {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
				keyPhase, deployItem.Status.DeployItemPhase,
				keyPhaseDeployer, deployItem.Status.Phase,
				keyJobID, deployItem.Status.JobID,
				keyJobIDFinished, deployItem.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.log.Error(err, msg,
				keyWriteID, writeID,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
				keyPhase, deployItem.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}
