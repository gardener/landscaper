package lib

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/landscaper/targetresolver"
	"github.com/gardener/landscaper/pkg/utils/read_write_layer"
)

// GetTargetClient is used to determine the client to read resources for custom readiness checks, export collection, and deletion groups.
// Usually it is the client obtained from the Target of the DeployItem.
// In some scenarios however, the Landscaper deploys an installer on a primary target cluster, and the installer
// deploys the actual application on a secondary target cluster.
func GetTargetClient(
	ctx context.Context,
	primaryTargetClient client.Client,
	lsClient client.Client,
	namespace string,
	secondaryTargetName *string) (targetClient client.Client, err error) {

	if secondaryTargetName == nil {
		return primaryTargetClient, nil
	}

	if lsClient == nil {
		return nil, fmt.Errorf("unable to get secondary target %s, because lsClient is not initialized", *secondaryTargetName)
	}

	target := &lsv1alpha1.Target{}
	targetKey := client.ObjectKey{
		Name:      *secondaryTargetName,
		Namespace: namespace,
	}
	err = read_write_layer.GetTarget(ctx, lsClient, targetKey, target, read_write_layer.R000005)
	if err != nil {
		return nil, fmt.Errorf("unable to read secondary target %s: %w", *secondaryTargetName, err)
	}

	resolvedTarget, err := targetresolver.Resolve(ctx, target, lsClient)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve secondary target %s: %w", *secondaryTargetName, err)
	}

	_, targetClient, _, err = GetClientMud(ctx, resolvedTarget, lsClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get secondary target client %s: %w", *secondaryTargetName, err)
	}

	return targetClient, nil
}
