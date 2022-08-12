package read_write_layer

import (
	"context"
	"fmt"

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

// methods for contexts

func (w *Writer) CreateOrPatchCoreContext(ctx context.Context, writeID WriteID, lsContext *lsv1alpha1.Context,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(lsContext)
	result, err := createOrPatchCore(ctx, w.client, lsContext, f)
	w.logContextUpdate(ctx, writeID, opContextCreateOrUpdate, lsContext, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

// methods for targets

func (w *Writer) CreateOrUpdateCoreTarget(ctx context.Context, writeID WriteID, target *lsv1alpha1.Target,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(target)
	result, err := createOrUpdateCore(ctx, w.client, target, f)
	w.logTargetUpdate(ctx, writeID, opTargetCreateOrUpdate, target, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

// methods for data objects

func (w *Writer) CreateOrUpdateCoreDataObject(ctx context.Context, writeID WriteID, do *lsv1alpha1.DataObject,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(do)
	result, err := createOrUpdateCore(ctx, w.client, do, f)
	w.logDataObjectUpdate(ctx, writeID, opDOCreateOrUpdate, do, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) CreateOrUpdateDataObject(ctx context.Context, writeID WriteID, do *lsv1alpha1.DataObject,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(do)
	result, err := kubernetes.CreateOrUpdate(ctx, w.client, do, f)
	w.logDataObjectUpdate(ctx, writeID, opDOCreateOrUpdate, do, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

// methods for installations

func (w *Writer) CreateOrUpdateInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	result, err := kubernetes.CreateOrUpdate(ctx, w.client, installation, f)
	w.logInstallationUpdate(ctx, writeID, opInstCreateOrUpdate, installation, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) CreateOrUpdateCoreInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	result, err := createOrUpdateCore(ctx, w.client, installation, f)
	w.logInstallationUpdate(ctx, writeID, opInstSpec, installation, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := update(ctx, w.client, installation)
	w.logInstallationUpdate(ctx, writeID, opInstSpec, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateInstallationStatus(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := updateStatus(ctx, w.client.Status(), installation)
	w.logInstallationUpdate(ctx, writeID, opInstStatus, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteInstallation(ctx context.Context, writeID WriteID, installation *lsv1alpha1.Installation) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(installation)
	err := delete(ctx, w.client, installation)
	w.logInstallationUpdate(ctx, writeID, opInstDelete, installation, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for executions

func (w *Writer) CreateOrUpdateExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	result, err := kubernetes.CreateOrUpdate(ctx, w.client, execution, f)
	w.logExecutionUpdate(ctx, writeID, opExecCreateOrUpdate, execution, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := update(ctx, w.client, execution)
	w.logExecutionUpdate(ctx, writeID, opExecSpec, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) PatchExecution(ctx context.Context, writeID WriteID, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(old)
	err := patch(ctx, w.client, new, client.MergeFrom(old), opts...)
	w.logExecutionUpdate(ctx, writeID, opExecSpec, new, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateExecutionStatus(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := updateStatus(ctx, w.client.Status(), execution)
	w.logExecutionUpdate(ctx, writeID, opExecStatus, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) PatchExecutionStatus(ctx context.Context, writeID WriteID, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(old)
	err := patchStatus(ctx, w.client.Status(), new, client.MergeFrom(old), opts...)
	w.logExecutionUpdate(ctx, writeID, opExecStatus, new, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteExecution(ctx context.Context, writeID WriteID, execution *lsv1alpha1.Execution) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(execution)
	err := delete(ctx, w.client, execution)
	w.logExecutionUpdate(ctx, writeID, opExecDelete, execution, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// methods for deploy items

func (w *Writer) CreateOrUpdateDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	result, err := kubernetes.CreateOrUpdate(ctx, w.client, deployItem, f)
	w.logDeployItemUpdate(ctx, writeID, opDICreateOrUpdate, deployItem, generationOld, resourceVersionOld, err)
	return result, errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := update(ctx, w.client, deployItem)
	w.logDeployItemUpdate(ctx, writeID, opDISpec, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) UpdateDeployItemStatus(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := updateStatus(ctx, w.client.Status(), deployItem)
	w.logDeployItemUpdate(ctx, writeID, opDIStatus, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) PatchDeployItemStatus(ctx context.Context, writeID WriteID, new *lsv1alpha1.DeployItem, old *lsv1alpha1.DeployItem,
	opts ...client.PatchOption) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(old)
	err := patchStatus(ctx, w.client.Status(), new, client.MergeFrom(old), opts...)
	w.logDeployItemUpdate(ctx, writeID, opDIStatus, new, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

func (w *Writer) DeleteDeployItem(ctx context.Context, writeID WriteID, deployItem *lsv1alpha1.DeployItem) error {
	generationOld, resourceVersionOld := getGenerationAndResourceVersion(deployItem)
	err := delete(ctx, w.client, deployItem)
	w.logDeployItemUpdate(ctx, writeID, opDIDelete, deployItem, generationOld, resourceVersionOld, err)
	return errorWithWriteID(err, writeID)
}

// base methods

func createOrPatchCore(ctx context.Context, c client.Client, object client.Object,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrPatch(ctx, c, object, f)
}

func createOrUpdateCore(ctx context.Context, c client.Client, object client.Object,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, c, object, f)
}

func update(ctx context.Context, c client.Client, object client.Object) error {
	return c.Update(ctx, object)
}

func patch(ctx context.Context, c client.Client, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.Patch(ctx, object, patch, opts...)
}

func updateStatus(ctx context.Context, c client.StatusWriter, object client.Object) error {
	return c.Update(ctx, object)
}

func patchStatus(ctx context.Context, c client.StatusWriter, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.Patch(ctx, object, patch, opts...)
}

func delete(ctx context.Context, c client.Client, object client.Object) error {
	return c.Delete(ctx, object)
}

func getGenerationAndResourceVersion(object client.Object) (generation int64, resourceVersion string) {
	generation = object.GetGeneration()
	resourceVersion = object.GetResourceVersion()
	return
}

func errorWithWriteID(err error, writeID WriteID) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("write operation %s failed: %w", writeID, err)
}
