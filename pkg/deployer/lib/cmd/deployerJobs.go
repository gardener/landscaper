package cmd

import "context"

type DeployerJob interface {
	StartDeployerJob(ctx context.Context) error
}
