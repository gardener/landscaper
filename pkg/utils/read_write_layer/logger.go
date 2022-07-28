package read_write_layer

import (
	"context"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/gardener/landscaper/pkg/utils"
)

const (
	historyLogLevel       logging.LogLevel = logging.DEBUG
	keyName                                = "name"
	keyNamespace                           = "namespace"
	keyPhase                               = "phase"
	keyPhaseDeployer                       = "phase-d"
	keyWriteID                             = "write-id"
	keyGenerationOld                       = "gen-old"
	keyGenerationNew                       = "gen-new"
	keyOperation                           = "op"
	keyResourceVersionOld                  = "rv-old"
	keyResourceVersionNew                  = "rv-new"
	keyJobID                               = "j-id"
	keyJobIDFinished                       = "j-id-f"
)

// getLogger tries to fetch the most up-to-date logger from the context
// and falls back to the writer's logger if that fails.
// The keys and values are only added in case of the fallback.
func (w *Writer) getLogger(ctx context.Context, keysAndValues ...interface{}) logging.Logger {
	log, err := logging.FromContext(ctx)
	if err != nil {
		return w.log.WithValues(keysAndValues...)
	}
	return log
}

func (w *Writer) logContextUpdate(ctx context.Context, writeID WriteID, msg string, con *lsv1alpha1.Context,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(con)
		w.getLogger(ctx,
			keyName, con.Name,
			keyNamespace, con.Namespace,
		).Log(historyLogLevel, msg,
			keyWriteID, writeID,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx,
			keyName, con.Name,
			keyNamespace, con.Namespace,
		).Error(err, msg,
			keyWriteID, writeID,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logTargetUpdate(ctx context.Context, writeID WriteID, msg string, target *lsv1alpha1.Target,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(target)
		w.getLogger(ctx,
			keyName, target.Name,
			keyNamespace, target.Namespace,
		).Log(historyLogLevel, msg,
			keyWriteID, writeID,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx,
			keyName, target.Name,
			keyNamespace, target.Namespace,
		).Error(err, msg,
			keyWriteID, writeID,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDataObjectUpdate(ctx context.Context, writeID WriteID, msg string, do *lsv1alpha1.DataObject,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(do)
		w.getLogger(ctx,
			keyName, do.Name,
			keyNamespace, do.Namespace,
		).Log(historyLogLevel, msg,
			keyWriteID, writeID,
			keyGenerationOld, generationOld,
			keyGenerationNew, generationNew,
			keyResourceVersionOld, resourceVersionOld,
			keyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx).Error(err, msg,
			keyWriteID, writeID,
			keyName, do.Name,
			keyNamespace, do.Namespace,
			keyGenerationOld, generationOld,
			keyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logInstallationUpdate(ctx context.Context, writeID WriteID, msg string, installation *lsv1alpha1.Installation,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(installation)
		opNew := lsv1alpha1helper.GetOperation(installation.ObjectMeta)
		if utils.IsNewReconcile() {
			w.getLogger(ctx,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, installation.Status.InstallationPhase,
				keyJobID, installation.Status.JobID,
				keyJobIDFinished, installation.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.getLogger(ctx,
				keyName, installation.Name,
				keyNamespace, installation.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, installation.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}

func (w *Writer) logExecutionUpdate(ctx context.Context, writeID WriteID, msg string, execution *lsv1alpha1.Execution,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(execution)
		opNew := lsv1alpha1helper.GetOperation(execution.ObjectMeta)
		if utils.IsNewReconcile() {
			w.getLogger(ctx,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, execution.Status.ExecutionPhase,
				keyJobID, execution.Status.JobID,
				keyJobIDFinished, execution.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.getLogger(ctx,
				keyName, execution.Name,
				keyNamespace, execution.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, execution.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}

func (w *Writer) logDeployItemUpdate(ctx context.Context, writeID WriteID, msg string, deployItem *lsv1alpha1.DeployItem,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(deployItem)
		opNew := lsv1alpha1helper.GetOperation(deployItem.ObjectMeta)
		if utils.IsNewReconcile() {
			w.getLogger(ctx,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
			).Log(historyLogLevel, msg,
				keyWriteID, writeID,
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
			w.getLogger(ctx,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, deployItem.Status.DeployItemPhase,
				keyPhaseDeployer, deployItem.Status.Phase,
				keyJobID, deployItem.Status.JobID,
				keyJobIDFinished, deployItem.Status.JobIDFinished,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		} else {
			w.getLogger(ctx,
				keyName, deployItem.Name,
				keyNamespace, deployItem.Namespace,
			).Error(err, msg,
				keyWriteID, writeID,
				keyPhase, deployItem.Status.Phase,
				keyGenerationOld, generationOld,
				keyResourceVersionOld, resourceVersionOld,
			)
		}
	}
}
