package read_write_layer

import (
	"context"
	"fmt"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	historyLogLevel          logging.LogLevel = logging.INFO
	keySecondDeployItemPhase                  = "landscaperDeployItemPhase"
)

// getLogger tries to fetch the most up-to-date logger from the context
// and falls back to creating a new one if that fails.
// The keys and values are only added in case of the fallback.
func (w *Writer) getLogger(ctx context.Context, keysAndValues ...interface{}) logging.Logger {
	log, _ := logging.FromContextOrNew(ctx, keysAndValues)
	return log
}

func (w *Writer) logContextUpdate(ctx context.Context, writeID WriteID, msg string, con *lsv1alpha1.Context,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(con)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", con.Namespace, con.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", con.Namespace, con.Name),
		).Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logTargetUpdate(ctx context.Context, writeID WriteID, msg string, target *lsv1alpha1.Target,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(target)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", target.Namespace, target.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", target.Namespace, target.Name),
		).Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDataObjectUpdate(ctx context.Context, writeID WriteID, msg string, do *lsv1alpha1.DataObject,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(do)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", do.Namespace, do.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx).Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyResource, fmt.Sprintf("%s/%s", do.Namespace, do.Name),
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logInstallationUpdate(ctx context.Context, writeID WriteID, msg string, installation *lsv1alpha1.Installation,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(installation)
		opNew := lsv1alpha1helper.GetOperation(installation.ObjectMeta)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", installation.Namespace, installation.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyInstallationPhase, installation.Status.InstallationPhase,
			lc.KeyJobID, installation.Status.JobID,
			lc.KeyJobIDFinished, installation.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyOperation, opNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)

	} else {
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", installation.Namespace, installation.Name),
		).Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyInstallationPhase, installation.Status.InstallationPhase,
			lc.KeyJobID, installation.Status.JobID,
			lc.KeyJobIDFinished, installation.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logExecutionUpdate(ctx context.Context, writeID WriteID, msg string, execution *lsv1alpha1.Execution,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(execution)
		opNew := lsv1alpha1helper.GetOperation(execution.ObjectMeta)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", execution.Namespace, execution.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyExecutionPhase, execution.Status.ExecutionPhase,
			lc.KeyJobID, execution.Status.JobID,
			lc.KeyJobIDFinished, execution.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyOperation, opNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else {
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", execution.Namespace, execution.Name),
		).Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyExecutionPhase, execution.Status.ExecutionPhase,
			lc.KeyJobID, execution.Status.JobID,
			lc.KeyJobIDFinished, execution.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDeployItemUpdate(ctx context.Context, writeID WriteID, msg string, deployItem *lsv1alpha1.DeployItem,
	generationOld int64, resourceVersionOld string, err error) {
	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(deployItem)
		opNew := lsv1alpha1helper.GetOperation(deployItem.ObjectMeta)
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", deployItem.Namespace, deployItem.Name),
		).Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			keySecondDeployItemPhase, deployItem.Status.DeployItemPhase,
			lc.KeyDeployItemPhase, deployItem.Status.Phase,
			lc.KeyJobID, deployItem.Status.GetJobID(),
			lc.KeyJobIDFinished, deployItem.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyOperation, opNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)

	} else {
		w.getLogger(ctx,
			lc.KeyResource, fmt.Sprintf("%s/%s", deployItem.Namespace, deployItem.Name),
		).Error(err, msg,
			lc.KeyWriteID, writeID,
			keySecondDeployItemPhase, deployItem.Status.DeployItemPhase,
			lc.KeyDeployItemPhase, deployItem.Status.Phase,
			lc.KeyJobID, deployItem.Status.GetJobID(),
			lc.KeyJobIDFinished, deployItem.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}
