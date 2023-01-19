package lib

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

type InterruptionChecker struct {
	deployItem *lsv1alpha1.DeployItem
	lsClient   client.Client
}

func NewInterruptionChecker(deployItem *lsv1alpha1.DeployItem, lsClient client.Client) *InterruptionChecker {
	return &InterruptionChecker{
		deployItem: deployItem,
		lsClient:   lsClient,
	}
}

func (c *InterruptionChecker) Check(ctx context.Context) error {
	if c == nil {
		return nil
	}

	di := &lsv1alpha1.DeployItem{}
	err := read_write_layer.GetDeployItem(ctx, c.lsClient, client.ObjectKeyFromObject(c.deployItem), di)
	if err != nil {
		return err
	}

	if di.Status.Phase == lsv1alpha1.DeployItemPhases.Failed {
		return fmt.Errorf("interrupted during readiness check/export collection")
	}

	return nil
}
