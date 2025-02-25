package rbac

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallRBAC(ctx context.Context, resourceClient client.Client, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	if err := resources.CreateOrUpdateResource(ctx, resourceClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return err
	}

	if valHelper.isCreateServiceAccount() {
		// Create or update RBAC objects for the landscaper controller
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newControllerClusterRoleMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newControllerServiceAccountMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newControllerClusterRoleBindingMutator(valHelper)); err != nil {
			return err
		}

		// Create or update RBAC objects for the landscaper user
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newUserClusterRoleMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newUserServiceAccountMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newUserClusterRoleBindingMutator(valHelper)); err != nil {
			return err
		}

		// Create or update RBAC objects for the landscaper webhooks
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newWebhooksClusterRoleMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newWebhooksServiceAccountMutator(valHelper)); err != nil {
			return err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newWebhooksClusterRoleBindingMutator(valHelper)); err != nil {
			return err
		}
	}

	return nil
}

func UninstallManifestDeployer(ctx context.Context, resourceClient client.Client, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	// Delete RBAC objects for the landscaper webhooks
	if err := resources.DeleteResource(ctx, resourceClient, newWebhooksClusterRoleBindingMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newWebhooksServiceAccountMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newWebhooksClusterRoleMutator(valHelper)); err != nil {
		return err
	}

	// Delete RBAC objects for the landscaper user
	if err := resources.DeleteResource(ctx, resourceClient, newUserClusterRoleBindingMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newUserServiceAccountMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newUserClusterRoleMutator(valHelper)); err != nil {
		return err
	}

	// Delete RBAC objects for the landscaper controller
	if err := resources.DeleteResource(ctx, resourceClient, newControllerClusterRoleBindingMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newControllerServiceAccountMutator(valHelper)); err != nil {
		return err
	}
	if err := resources.DeleteResource(ctx, resourceClient, newControllerClusterRoleMutator(valHelper)); err != nil {
		return err
	}

	return nil
}
