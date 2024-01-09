package read_write_layer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/landscaper/apis/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

type Writer struct {
	client client.Client
}

func NewWriter(c client.Client) *Writer {
	return &Writer{
		client: c,
	}
}

// methods for sync objects

func (w *Writer) CreateSyncObject(ctx context.Context, writeID WriteID, syncObject *lsv1alpha1.SyncObject) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(syncObject)
	err := create(ctx, w.client, syncObject, writeID, opSyncObjectCreate)
	w.logSyncObjectUpdateBasic(ctx, writeID, opSyncObjectCreate, syncObject, generationOld, resourceVersionOld, err, true)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateSyncObject(ctx context.Context, writeID WriteID, syncObject *lsv1alpha1.SyncObject) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(syncObject)
	err := update(ctx, w.client, syncObject, writeID, opSyncObjectSpec)
	w.logSyncObjectUpdate(ctx, writeID, opSyncObjectSpec, syncObject, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteSyncObject(ctx context.Context, writeID WriteID, syncObject *lsv1alpha1.SyncObject) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(syncObject)
	err := delete(ctx, w.client, syncObject, writeID, opSyncObjectDelete)
	w.logSyncObjectUpdate(ctx, writeID, opSyncObjectDelete, syncObject, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for contexts

func (w *Writer) CreateOrPatchCoreContext(ctx context.Context, writeID WriteID, lsContext *lsv1alpha1.Context,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(lsContext)
	result, err := createOrPatchCore(ctx, w.client, lsContext, f, writeID, opContextCreateOrUpdate)
	w.logContextUpdate(ctx, writeID, opContextCreateOrUpdate, lsContext, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

// methods for targets

func (w *Writer) CreateOrUpdateCoreTarget(ctx context.Context, writeID WriteID, target *lsv1alpha1.Target,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(target)
	result, err := createOrUpdateCore(ctx, w.client, target, f, writeID, opTargetCreateOrUpdate)
	w.logTargetUpdate(ctx, writeID, opTargetCreateOrUpdate, target, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteTarget(ctx context.Context, writeID WriteID, target *lsv1alpha1.Target) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(target)
	err := delete(ctx, w.client, target, writeID, opTargetDelete)
	w.logTargetUpdate(ctx, writeID, opTargetDelete, target, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for data objects

func (w *Writer) CreateOrUpdateCoreDataObject(ctx context.Context, writeID WriteID, do *lsv1alpha1.DataObject,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(do)
	result, err := createOrUpdateCore(ctx, w.client, do, f, writeID, opDOCreateOrUpdate)
	w.logDataObjectUpdate(ctx, writeID, opDOCreateOrUpdate, do, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) CreateOrUpdateDataObject(ctx context.Context, writeID WriteID, do *lsv1alpha1.DataObject,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(do)
	result, err := createOrUpdateKubernetes(ctx, w.client, do, f, writeID, opDOCreateOrUpdate)
	w.logDataObjectUpdate(ctx, writeID, opDOCreateOrUpdate, do, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteDataObject(ctx context.Context, writeID WriteID, do *lsv1alpha1.DataObject) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(do)
	err := delete(ctx, w.client, do, writeID, opInstDelete)
	w.logDataObjectUpdate(ctx, writeID, opInstDelete, do, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for installations

func (w *Writer) CreateOrUpdateInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	result, err := createOrUpdateKubernetes(ctx, w.client, installation, f, writeID, opInstCreateOrUpdate)
	w.logInstallationUpdate(ctx, writeID, opInstCreateOrUpdate, installation, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) CreateOrUpdateCoreInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	result, err := createOrUpdateCore(ctx, w.client, installation, f, writeID, opInstSpec)
	w.logInstallationUpdate(ctx, writeID, opInstSpec, installation, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := update(ctx, w.client, installation, writeID, opInstSpec)
	w.logInstallationUpdate(ctx, writeID, opInstSpec, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateInstallationStatus(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := updateStatus(ctx, w.client.Status(), installation, writeID, opInstStatus)
	w.logInstallationUpdate(ctx, writeID, opInstStatus, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := delete(ctx, w.client, installation, writeID, opInstDelete)
	w.logInstallationUpdate(ctx, writeID, opInstDelete, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for executions

func (w *Writer) CreateOrUpdateExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	result, err := createOrUpdateKubernetes(ctx, w.client, execution, f, writeID, opExecCreateOrUpdate)
	w.logExecutionUpdate(ctx, writeID, opExecCreateOrUpdate, execution, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := update(ctx, w.client, execution, writeID, opExecSpec)
	w.logExecutionUpdate(ctx, writeID, opExecSpec, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateExecutionStatus(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := updateStatus(ctx, w.client.Status(), execution, writeID, opExecStatus)
	w.logExecutionUpdate(ctx, writeID, opExecStatus, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := delete(ctx, w.client, execution, writeID, opExecDelete)
	w.logExecutionUpdate(ctx, writeID, opExecDelete, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for deploy items

func (w *Writer) CreateOrUpdateDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	result, err := createOrUpdateKubernetes(ctx, w.client, deployItem, f, writeID, opDICreateOrUpdate)
	w.logDeployItemUpdate(ctx, writeID, opDICreateOrUpdate, deployItem, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := update(ctx, w.client, deployItem, writeID, opDISpec)
	w.logDeployItemUpdate(ctx, writeID, opDISpec, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateDeployItemStatus(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := updateStatus(ctx, w.client.Status(), deployItem, writeID, opDIStatus)
	w.logDeployItemUpdate(ctx, writeID, opDIStatus, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := delete(ctx, w.client, deployItem, writeID, opDIDelete)
	w.logDeployItemUpdate(ctx, writeID, opDIDelete, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// base methods

func create(ctx context.Context, c client.Client, object client.Object, writeID WriteID, msg string) error {
	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.Create(ctx, object)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return err
}

func createOrUpdateKubernetes(ctx context.Context, c client.Client, object client.Object,
	f controllerutil.MutateFn, writeID WriteID, msg string) (controllerutil.OperationResult, error) {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	or, err := kubernetes.CreateOrUpdate(ctx, c, object, f)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return or, err
}

func createOrPatchCore(ctx context.Context, c client.Client, object client.Object,
	f controllerutil.MutateFn, writeID WriteID, msg string) (controllerutil.OperationResult, error) {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	or, err := controllerutil.CreateOrPatch(ctx, c, object, f)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return or, err
}

func createOrUpdateCore(ctx context.Context, c client.Client, object client.Object,
	f controllerutil.MutateFn, writeID WriteID, msg string) (controllerutil.OperationResult, error) {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	or, err := controllerutil.CreateOrUpdate(ctx, c, object, f)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return or, err
}

func update(ctx context.Context, c client.Client, object client.Object, writeID WriteID, msg string) error {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.Update(ctx, object)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return err
}

func updateStatus(ctx context.Context, c client.StatusWriter, object client.Object, writeID WriteID, msg string) error {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.Update(ctx, object)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return err
}

func delete(ctx context.Context, c client.Client, object client.Object, writeID WriteID, msg string) error {

	log, ctx := logging.FromContextOrNew(ctx, nil,
		keyFetchedResource, fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName()),
		lc.KeyWriteID, writeID)

	debugEnabled := log.Enabled(logging.DEBUG)
	var start time.Time
	if debugEnabled {
		start = time.Now()
	}

	err := c.Delete(ctx, object)

	if debugEnabled {
		if err != nil {
			log = log.WithValues(lc.KeyError, err.Error())
		}

		duration := time.Since(start).Milliseconds()
		if duration > 1000 {
			msg = msg + " - duration: " + strconv.FormatInt(duration, 10)
		}
		log.Debug(msg)
	}

	return err
}

func getGenerationAndResourceVersion(object client.Object) (generation int64, resourceVersion string) {
	generation = object.GetGeneration()
	resourceVersion = object.GetResourceVersion()
	return
}

func errorWithWriteID(err error, writeID WriteID) errors.LsError {
	if err == nil {
		return nil
	}

	errorCodes := []lsv1alpha1.ErrorCode{}
	if isRecoverableError(err) {
		errorCodes = append(errorCodes, lsv1alpha1.ErrorWebhook)
	}

	if apierrors.IsConflict(err) {
		errorCodes = append(errorCodes, lsv1alpha1.ErrorForInfoOnly)
	}

	lsError := errors.NewWrappedError(err, "errorWithWriteID", "write",
		fmt.Sprintf("write operation %s failed with %s", writeID, err.Error()), errorCodes...)

	return lsError
}

func isRecoverableError(err error) bool {
	// There are sometimes intermediate problems with the webhook preventing a write operation.
	// Such errors should result in a retry of the reconcile operation.
	isWebhookProblem := strings.Contains(err.Error(), "webhook")
	isSpecialWebhookProblem := strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "failed to call webhook") ||
		strings.Contains(err.Error(), "proxy error")

	return isWebhookProblem && isSpecialWebhookProblem
}
