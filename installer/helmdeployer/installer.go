package helmdeployer

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper/installer/resources"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func InstallHelmDeployer(ctx context.Context, hostCluster *resources.Cluster, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	hostClient := hostCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, hostClient, resources.NewNamespaceMutator(valHelper.hostNamespace())); err != nil {
		return err
	}

	if valHelper.isCreateServiceAccount() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newClusterRoleMutator(valHelper)); err != nil {
			return err
		}

		if err := resources.CreateOrUpdateResource(ctx, hostClient, newServiceAccountMutator(valHelper)); err != nil {
			return err
		}

		if err := resources.CreateOrUpdateResource(ctx, hostClient, newClusterRoleBindingMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if len(valHelper.landscaperClusterKubeconfig()) > 0 {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newKubeconfigSecretMutator(valHelper)); err != nil {
			return err
		}
	}

	if valHelper.values.OCI != nil {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newRegistrySecretMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newDeploymentMutator(valHelper)); err != nil {
		return err
	}

	return nil
}

func UninstallHelmDeployer(ctx context.Context, hostCluster *resources.Cluster, values *Values) error {

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

func newHostClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig for host cluster of helm deployer: %v\n", err)
	}
	hostClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client for host cluster of helm deployer: %v\n", err)
	}
	return hostClient, nil
}
