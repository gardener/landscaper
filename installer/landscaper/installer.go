package landscaper

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallLandscaper(ctx context.Context, hostClient client.Client, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

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

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newServiceMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, hostClient, newWebhooksServiceMutator(valHelper)); err != nil {
		return err
	}

	if valHelper.values.WebhooksServer.Ingress != nil {
		if err := resources.CreateOrUpdateResource(ctx, hostClient, newIngressMutator(valHelper)); err != nil {
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

func UninstallLandscaper(ctx context.Context, hostClient client.Client, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newWebhooksHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newCentralHPAMutator(valHelper)); err != nil {
		return err
	}

	if err := resources.DeleteResource(ctx, hostClient, newMainHPAMutator(valHelper)); err != nil {
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
