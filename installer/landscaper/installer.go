package landscaper

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
)

func InstallLandscaper(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	hostClient := values.HostCluster.Client()

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

	if len(valHelper.values.Controller.LandscaperKubeconfig.Kubeconfig) > 0 {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newControllerKubeconfigSecretMutator(valHelper)); err != nil {
			return err
		}
	}

	if len(valHelper.values.WebhooksServer.LandscaperKubeconfig.Kubeconfig) > 0 {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksServiceMutator(valHelper)); err != nil {
			return err
		}
	}

	if valHelper.values.WebhooksServer.Ingress != nil {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newIngressMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if !valHelper.areAllWebhooksDisabled() {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
			return err
		}
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	return nil
}

func UninstallLandscaper(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelperForDelete(values)
	if err != nil {
		return err
	}

	hostClient := values.HostCluster.Client()

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newMainHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newMainDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newCentralDeploymentMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newIngressMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newConfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksKubeconfigSecretMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newControllerKubeconfigSecretMutator(valHelper)); err != nil {
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
