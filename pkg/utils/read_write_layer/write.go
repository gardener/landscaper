package read_write_layer

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
)

// write methods create, update, patch, delete

// methods for installations

func CreateOrUpdateInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	hasName := installation.Name != ""
	equal := false
	var externalErr error = nil
	if hasName {
		equal, externalErr = installationEqual(ctx, c, installation)
		if externalErr != nil && !apierrors.IsNotFound(externalErr) {
			return controllerutil.OperationResultNone, externalErr
		}
	}

	return kubernetes.CreateOrUpdate(ctx, c, installation, func() error {
		if err := f(); err != nil {
			return err
		}

		if !equal || externalErr != nil {
			return addHistoryItemToInstallation(writeID, installation)
		}

		return nil
	})
}

func CreateOrUpdateCoreInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	hasName := installation.Name != ""
	equal := false
	var externalErr error = nil
	if hasName {
		equal, externalErr = installationEqual(ctx, c, installation)
		if externalErr != nil && !apierrors.IsNotFound(externalErr) {
			return controllerutil.OperationResultNone, externalErr
		}
	}

	return createOrUpdateCore(ctx, c, installation, func() error {
		if err := f(); err != nil {
			return err
		}

		if !equal || externalErr != nil {
			return addHistoryItemToInstallation(writeID, installation)
		}

		return nil
	})
}

func UpdateInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation) error {
	equal, externalErr := installationEqual(ctx, c, installation)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		if err := addHistoryItemToInstallation(writeID, installation); err != nil {
			return err
		}
		return update(ctx, c, installation)
	}

	return nil
}

func UpdateInstallationStatus(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation) error {
	equal, externalErr := installationStatusEqual(ctx, c, installation)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		addHistoryItemToInstallationStatus(writeID, installation)
		return updateStatus(ctx, c, installation)
	}
	return nil
}

func DeleteInstallation(ctx context.Context, writeID WriteID, c client.Client, installation *lsv1alpha1.Installation) error {
	return delete(ctx, c, installation)
}

// methods for executions
func CreateOrUpdateExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	equal, externalErr := executionEqual(ctx, c, execution)
	if externalErr != nil && !apierrors.IsNotFound(externalErr) {
		return controllerutil.OperationResultNone, externalErr
	}

	return kubernetes.CreateOrUpdate(ctx, c, execution, func() error {
		if err := f(); err != nil {
			return err
		}

		if !equal || externalErr != nil {
			return addHistoryItemToExecution(writeID, execution)
		}

		return nil
	})
}

func UpdateExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution) error {
	equal, externalErr := executionEqual(ctx, c, execution)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		if err := addHistoryItemToExecution(writeID, execution); err != nil {
			return err
		}
		return update(ctx, c, execution)
	}

	return nil
}

func PatchExecution(ctx context.Context, writeID WriteID, c client.Client, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	if err := addHistoryItemToExecution(writeID, new); err != nil {
		return err
	}

	return patch(ctx, c, new, client.MergeFrom(old), opts...)
}

func UpdateExecutionStatus(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution) error {
	equal, externalErr := executionStatusEqual(ctx, c, execution)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		addHistoryItemToExecutionStatus(writeID, execution)
		return updateStatus(ctx, c, execution)
	}
	return nil
}

func PatchExecutionStatus(ctx context.Context, writeID WriteID, c client.Client, new *lsv1alpha1.Execution, old *lsv1alpha1.Execution,
	opts ...client.PatchOption) error {
	addHistoryItemToExecutionStatus(writeID, new)
	return patchStatus(ctx, c, new, client.MergeFrom(old), opts...)
}

func DeleteExecution(ctx context.Context, writeID WriteID, c client.Client, execution *lsv1alpha1.Execution) error {
	return delete(ctx, c, execution)
}

// methods for deploy items
func CreateOrUpdateDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem,
	f controllerutil.MutateFn) (controllerutil.OperationResult, error) {

	hasName := deployItem.Name != ""
	equal := false
	var externalErr error = nil
	if hasName {
		equal, externalErr = deployItemEqual(ctx, c, deployItem)
		if externalErr != nil && !apierrors.IsNotFound(externalErr) {
			return controllerutil.OperationResultNone, externalErr
		}
	}

	return kubernetes.CreateOrUpdate(ctx, c, deployItem, func() error {
		if err := f(); err != nil {
			return err
		}

		if !equal || externalErr != nil {
			return addHistoryItemToDeployItem(writeID, deployItem)
		}

		return nil
	})
}

func UpdateDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem) error {
	equal, externalErr := deployItemEqual(ctx, c, deployItem)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		if err := addHistoryItemToDeployItem(writeID, deployItem); err != nil {
			return err
		}

		return update(ctx, c, deployItem)
	}
	return nil
}

func UpdateDeployItemStatus(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem) error {
	equal, externalErr := deployItemStatusEqual(ctx, c, deployItem)
	if externalErr != nil {
		return externalErr
	}

	if !equal {
		addHistoryItemToDeployItemStatus(writeID, deployItem)
		return updateStatus(ctx, c, deployItem)
	}
	return nil
}

func PatchDeployItemStatus(ctx context.Context, writeID WriteID, c client.Client, new *lsv1alpha1.DeployItem, old *lsv1alpha1.DeployItem,
	opts ...client.PatchOption) error {
	addHistoryItemToDeployItemStatus(writeID, new)
	return patchStatus(ctx, c, new, client.MergeFrom(old), opts...)
}

func DeleteDeployItem(ctx context.Context, writeID WriteID, c client.Client, deployItem *lsv1alpha1.DeployItem) error {
	return delete(ctx, c, deployItem)
}

// base methods
func installationEqual(ctx context.Context, c client.Client, current *lsv1alpha1.Installation) (bool, error) {
	old := lsv1alpha1.Installation{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.ObjectMeta, old.ObjectMeta) && reflect.DeepEqual(current.Spec, old.Spec)
	return equal, nil
}

func executionEqual(ctx context.Context, c client.Client, current *lsv1alpha1.Execution) (bool, error) {
	old := lsv1alpha1.Execution{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.ObjectMeta, old.ObjectMeta) && reflect.DeepEqual(current.Spec, old.Spec)
	return equal, nil
}

func deployItemEqual(ctx context.Context, c client.Client, current *lsv1alpha1.DeployItem) (bool, error) {
	old := lsv1alpha1.DeployItem{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.ObjectMeta, old.ObjectMeta) && reflect.DeepEqual(current.Spec, old.Spec)
	return equal, nil
}

func installationStatusEqual(ctx context.Context, c client.Client, current *lsv1alpha1.Installation) (bool, error) {
	old := lsv1alpha1.Installation{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.Status, old.Status)
	return equal, nil
}

func executionStatusEqual(ctx context.Context, c client.Client, current *lsv1alpha1.Execution) (bool, error) {
	old := lsv1alpha1.Execution{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.Status, old.Status)
	return equal, nil
}

func deployItemStatusEqual(ctx context.Context, c client.Client, current *lsv1alpha1.DeployItem) (bool, error) {
	old := lsv1alpha1.DeployItem{}
	err := c.Get(ctx, kubernetes.ObjectKeyFromObject(current), &old)
	if err != nil {
		return false, err
	}

	equal := reflect.DeepEqual(current.Status, old.Status)
	return equal, nil
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

func updateStatus(ctx context.Context, c client.Client, object client.Object) error {
	return c.Status().Update(ctx, object)
}

func patchStatus(ctx context.Context, c client.Client, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.Status().Patch(ctx, object, patch, opts...)
}

func delete(ctx context.Context, c client.Client, object client.Object) error {
	return c.Delete(ctx, object)
}
