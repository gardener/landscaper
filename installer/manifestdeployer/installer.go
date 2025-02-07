package manifestdeployer

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func InstallManifestDeployer(ctx context.Context, values *Values) error {

	hostClient, err := newHostClient()
	if err != nil {
		return err
	}

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	if err := CreateResource(ctx, hostClient, newNamespaceDefinition(valHelper)); err != nil {
		return err
	}

	if valHelper.isCreateServiceAccount() {
		if err := CreateResource(ctx, hostClient, newClusterRoleDefinition(valHelper)); err != nil {
			return err
		}

		if err := CreateResource(ctx, hostClient, newServiceAccountDefinition(valHelper)); err != nil {
			return err
		}

		if err := CreateResource(ctx, hostClient, newClusterRoleBindingDefinition(valHelper)); err != nil {
			return err
		}
	}

	if err := CreateResource(ctx, hostClient, newConfigSecretDefinition(valHelper)); err != nil {
		return err
	}

	if len(valHelper.landscaperClusterKubeconfig()) > 0 {
		if err := CreateResource(ctx, hostClient, newLandscaperClusterKubeconfigSecretDefinition(valHelper)); err != nil {
			return err
		}
	}

	if err := CreateResource(ctx, hostClient, newHPADefinition(valHelper)); err != nil {
		return err
	}

	if err := CreateResource(ctx, hostClient, newDeploymentDefinition(valHelper)); err != nil {
		return err
	}

	return nil
}

func UninstallManifestDeployer(ctx context.Context, values *Values) error {

	hostClient, err := newHostClient()
	if err != nil {
		return err
	}

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newDeploymentDefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newHPADefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newLandscaperClusterKubeconfigSecretDefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newConfigSecretDefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newClusterRoleBindingDefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newServiceAccountDefinition(valHelper)); err != nil {
		return err
	}

	if err := DeleteResource(ctx, hostClient, newClusterRoleDefinition(valHelper)); err != nil {
		return err
	}

	return nil
}

func newHostClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig for host cluster of manifest deployer: %v\n", err)
	}
	hostClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client for host cluster of manifest deployer: %v\n", err)
	}
	return hostClient, nil
}
