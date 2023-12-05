package read_write_layer

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	historyLogLevel logging.LogLevel = logging.INFO

	keyUpdatedResource = "updatedResource"
	keyFetchedResource = "fetchedResource"
)

// getLogger tries to fetch the most up-to-date logger from the context
// and falls back to creating a new one if that fails.
// The keys and values are only added in case of the fallback.
func (w *Writer) getLogger(ctx context.Context, keysAndValues ...interface{}) logging.Logger {
	log, _ := logging.FromContextOrNew(ctx, nil, keysAndValues...)
	return log
}

func (w *Writer) logContextUpdate(ctx context.Context, writeID WriteID, msg string, con *lsv1alpha1.Context,
	generationOld int64, resourceVersionOld string, err error) {
	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", con.Namespace, con.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(con)
		logger.Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logTargetUpdate(ctx context.Context, writeID WriteID, msg string, target *lsv1alpha1.Target,
	generationOld int64, resourceVersionOld string, err error) {

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", target.Namespace, target.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(target)
		logger.Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logSyncObjectUpdate(ctx context.Context, writeID WriteID, msg string, syncObject *lsv1alpha1.SyncObject,
	generationOld int64, resourceVersionOld string, err error) {

	w.logSyncObjectUpdateBasic(ctx, writeID, msg, syncObject, generationOld, resourceVersionOld, err, false)
}

func (w *Writer) logSyncObjectUpdateBasic(ctx context.Context, writeID WriteID, msg string,
	syncObject *lsv1alpha1.SyncObject, generationOld int64, resourceVersionOld string, err error, logAlreadyExistsAsInfo bool) {

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", syncObject.Namespace, syncObject.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(syncObject)
		logger.Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else if apierrors.IsConflict(err) || (logAlreadyExistsAsInfo && apierrors.IsAlreadyExists(err)) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logDataObjectUpdate(ctx context.Context, writeID WriteID, msg string, do *lsv1alpha1.DataObject,
	generationOld int64, resourceVersionOld string, err error) {

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", do.Namespace, do.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(do)
		logger.Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)
	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}

func (w *Writer) logInstallationUpdate(ctx context.Context, writeID WriteID, msg string, installation *lsv1alpha1.Installation,
	generationOld int64, resourceVersionOld string, err error) {

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", installation.Namespace, installation.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(installation)
		opNew := lsv1alpha1helper.GetOperation(installation.ObjectMeta)
		logger.Log(historyLogLevel, msg,
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
	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyInstallationPhase, installation.Status.InstallationPhase,
			lc.KeyJobID, installation.Status.JobID,
			lc.KeyJobIDFinished, installation.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
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

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", execution.Namespace, execution.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(execution)
		opNew := lsv1alpha1helper.GetOperation(execution.ObjectMeta)
		logger.Log(historyLogLevel, msg,
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
	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyExecutionPhase, execution.Status.ExecutionPhase,
			lc.KeyJobID, execution.Status.JobID,
			lc.KeyJobIDFinished, execution.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
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

	logger := w.getLogger(ctx, keyUpdatedResource, fmt.Sprintf("%s/%s", deployItem.Namespace, deployItem.Name))

	if err == nil {
		generationNew, resourceVersionNew := getGenerationAndResourceVersion(deployItem)
		opNew := lsv1alpha1helper.GetOperation(deployItem.ObjectMeta)
		logger.Log(historyLogLevel, msg,
			lc.KeyWriteID, writeID,
			lc.KeyDeployItemPhase, deployItem.Status.Phase,
			lc.KeyJobID, deployItem.Status.GetJobID(),
			lc.KeyJobIDFinished, deployItem.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyGenerationNew, generationNew,
			lc.KeyOperation, opNew,
			lc.KeyResourceVersionOld, resourceVersionOld,
			lc.KeyResourceVersionNew, resourceVersionNew,
		)

	} else if apierrors.IsConflict(err) {
		message := msg + ": " + err.Error()
		logger.Info(message,
			lc.KeyWriteID, writeID,
			lc.KeyDeployItemPhase, deployItem.Status.Phase,
			lc.KeyJobID, deployItem.Status.GetJobID(),
			lc.KeyJobIDFinished, deployItem.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	} else {
		logger.Error(err, msg,
			lc.KeyWriteID, writeID,
			lc.KeyDeployItemPhase, deployItem.Status.Phase,
			lc.KeyJobID, deployItem.Status.GetJobID(),
			lc.KeyJobIDFinished, deployItem.Status.JobIDFinished,
			lc.KeyGenerationOld, generationOld,
			lc.KeyResourceVersionOld, resourceVersionOld,
		)
	}
}
