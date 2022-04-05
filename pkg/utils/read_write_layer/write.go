package read_write_layer

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// write methods create, update, patch, delete

// methods for installations

func CreateOrUpdateInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return kubernetes.CreateOrUpdate(ctx, c, installation, f)
}

func CreateOrUpdateCoreInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return createOrUpdateCore(ctx, c, installation, f)
}

func UpdateInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation) error {
	//if err := addHistoryItemToInstallation(writeID, installation); err != nil {
	//	return err
	//}

	return update(ctx, c, installation)
}

func UpdateInstallationStatus(ctx context.Context, writeID WriteID, c client.StatusWriter, installation *lsv1alpha1.Installation) error {
	addHistoryItemToInstallationStatus(writeID, installation)
	return updateStatus(ctx, c, installation)
}

func DeleteInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation) error {
	return delete(ctx, c, installation)
}

// methods for executions
func CreateOrUpdateExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return kubernetes.CreateOrUpdate(ctx, c, execution, f)
}

func UpdateExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution) error {
	//if err := addHistoryItemToExecution(writeID, execution); err != nil {
	//	return err
	//}

	return update(ctx, c, execution)
}

func PatchExecution(ctx context.Context, writeID WriteID, c client.Client, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	//if err := addHistoryItemToExecution(writeID, new); err != nil {
	//	return err
	//}

	return patch(ctx, c, new, client.MergeFrom(old), opts...)
}

func UpdateExecutionStatus(ctx context.Context, writeID WriteID, c client.StatusWriter, execution *lsv1alpha1.Execution) error {
	addHistoryItemToExecutionStatus(writeID, execution)
	return updateStatus(ctx, c, execution)
}

func PatchExecutionStatus(ctx context.Context, writeID WriteID, c client.StatusWriter, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	//addHistoryItemToExecutionStatus(writeID, new)
	return patchStatus(ctx, c, new, client.MergeFrom(old), opts...)
}

func DeleteExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution) error {
	return delete(ctx, c, execution)
}

// methods for deploy items
func CreateOrUpdateDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	return kubernetes.CreateOrUpdate(ctx, c, deployItem, f)
}

func UpdateDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem) error {
	//if err := addHistoryItemToDeployItem(writeID, deployItem); err != nil {
	//	return err
	//}

	return update(ctx, c, deployItem)
}

func UpdateDeployItemStatus(ctx context.Context, writeID WriteID, c client.StatusWriter, deployItem *lsv1alpha1.DeployItem) error {
	addHistoryItemToDeployItemStatus(writeID, deployItem)
	return updateStatus(ctx, c, deployItem)
}

func PatchDeployItemStatus(ctx context.Context, writeID WriteID, c client.StatusWriter, new *lsv1alpha1.DeployItem, old *lsv1alpha1.DeployItem,
	opts ...client.PatchOption) error {
	//addHistoryItemToDeployItemStatus(writeID, new)
	return patchStatus(ctx, c, new, client.MergeFrom(old), opts...)
}

func DeleteDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem) error {
	return delete(ctx, c, deployItem)
}

// base methods
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
