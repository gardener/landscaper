package read_write_layer

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// read methods get, list

// read methods for installations
func GetInstallation(ctx context.Context, c client.Reader, key client.ObjectKey, installation *lsv1alpha1.Installation) error {
	return get(ctx, c, key, installation)
}

func ListInstallations(ctx context.Context, c client.Reader, installations *lsv1alpha1.InstallationList, opts ...client.ListOption) error {
	return list(ctx, c, installations, opts...)
}

// read methods for executions
func GetExecution(ctx context.Context, c client.Reader, key client.ObjectKey, execution *lsv1alpha1.Execution) error {
	return get(ctx, c, key, execution)
}

func ListExecutions(ctx context.Context, c client.Reader, executions *lsv1alpha1.ExecutionList, opts ...client.ListOption) error {
	return list(ctx, c, executions, opts...)
}

// read methods for deploy items
func GetDeployItem(ctx context.Context, c client.Reader, key client.ObjectKey, deployItem *lsv1alpha1.DeployItem) error {
	return get(ctx, c, key, deployItem)
}

func ListDeployItems(ctx context.Context, c client.Reader, deployItems *lsv1alpha1.DeployItemList, opts ...client.ListOption) error {
	return list(ctx, c, deployItems, opts...)
}

// basic functions
func get(ctx context.Context, c client.Reader, key client.ObjectKey, object client.Object) error {
	return c.Get(ctx, key, object)
}

func list(ctx context.Context, c client.Reader, objects client.ObjectList, opts ...client.ListOption) error {
	return c.List(ctx, objects, opts...)
}
