package interruption

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

type standardInterruptionChecker struct {
	deployItem *lsv1alpha1.DeployItem
	lsClient   client.Client
}

func NewStandardInterruptionChecker(deployItem *lsv1alpha1.DeployItem, lsClient client.Client) InterruptionChecker {
	return &standardInterruptionChecker{
		deployItem: deployItem,
		lsClient:   lsClient,
	}
}

func (c *standardInterruptionChecker) Check(ctx context.Context) error {
	if c == nil {
		return nil
	}

	di := &lsv1alpha1.DeployItem{}
	err := read_write_layer.GetDeployItem(ctx, c.lsClient, client.ObjectKeyFromObject(c.deployItem), di, read_write_layer.R000026)
	if err != nil {
		return err
	}

	if di.Status.Phase.IsFailed() {
		return ErrInterruption
	}

	return nil
}
