package manifestdeployer

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
)

type Exports struct {
	DeploymentName string
}

func InstallManifestDeployer(ctx context.Context, hostCluster *resources.Cluster, values *Values) (*Exports, error) {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return nil, err
	}

	hostClient := hostCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, hostClient, resources.NewNamespaceMutator(valHelper.hostNamespace())); err != nil {
		return nil, err
	}

	if valHelper.isCreateServiceAccount() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newClusterRoleMutator(valHelper)); err != nil {
			return nil, err
		}

		if err := resources.CreateOrUpdateResource(ctx, hostClient, newServiceAccountMutator(valHelper)); err != nil {
			return nil, err
		}

		if err := resources.CreateOrUpdateResource(ctx, hostClient, newClusterRoleBindingMutator(valHelper)); err != nil {
			return nil, err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return nil, err
	}

	if len(valHelper.landscaperClusterKubeconfig()) > 0 {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newKubeconfigSecretMutator(valHelper)); err != nil {
			return nil, err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newHPAMutator(valHelper)); err != nil {
		return nil, err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newDeploymentMutator(valHelper)); err != nil {
		return nil, err
	}

	return &Exports{
		// needed for health checks
		DeploymentName: valHelper.deployerFullName(),
	}, nil
}

func UninstallManifestDeployer(ctx context.Context, hostCluster *resources.Cluster, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	hostClient := hostCluster.Client()

	if err := resources.DeleteResource(ctx, hostClient, newDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newClusterRoleBindingMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newServiceAccountMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newClusterRoleMutator(valHelper)); err != nil {
		return err
	}

	return nil
}
