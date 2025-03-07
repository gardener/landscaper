package rbac

import (
	"context"
	"github.com/gardener/landscaper/installer/resources"
	core "k8s.io/api/core/v1"
)

type Kubeconfigs struct {
	ControllerKubeconfig []byte
	WebhooksKubeconfig   []byte
	UserKubeconfig       []byte
}

func InstallLandscaperRBACResources(ctx context.Context, values *Values) (kubeconfigs *Kubeconfigs, err error) {
	valHelper, err := newValuesHelper(values)
	if err != nil {
		return kubeconfigs, err
	}

	resourceClient := values.ResourceCluster.Client()

	if err := resources.CreateOrUpdateResource(ctx, resourceClient, resources.NewNamespaceMutator(valHelper.resourceNamespace())); err != nil {
		return kubeconfigs, err
	}

	if valHelper.isCreateServiceAccount() {
		// Create or update RBAC objects for the landscaper controller
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newControllerClusterRoleMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}
		controllerServiceAccountMutator := newControllerServiceAccountMutator(valHelper)
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, controllerServiceAccountMutator); err != nil {
			return kubeconfigs, err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newControllerClusterRoleBindingMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}

		// Create or update RBAC objects for the landscaper user
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newUserClusterRoleMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}
		userServiceAccountMutator := newUserServiceAccountMutator(valHelper)
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, resources.Mutator[*core.ServiceAccount](userServiceAccountMutator)); err != nil {
			return kubeconfigs, err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newUserClusterRoleBindingMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}

		// Create or update RBAC objects for the landscaper webhooks
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newWebhooksClusterRoleMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}
		webhooksServiceAccountMutator := newWebhooksServiceAccountMutator(valHelper)
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, webhooksServiceAccountMutator); err != nil {
			return kubeconfigs, err
		}
		if err := resources.CreateOrUpdateResource(ctx, resourceClient, newWebhooksClusterRoleBindingMutator(valHelper)); err != nil {
			return kubeconfigs, err
		}

		// Create kubeconfigs for the service accounts
		kubeconfigs = &Kubeconfigs{}

		controllerServiceAccount := newControllerServiceAccountMutator(valHelper).Empty()
		kubeconfigs.ControllerKubeconfig, err = resources.CreateKubeconfig(ctx, values.ResourceCluster, controllerServiceAccount.Name, controllerServiceAccount.Namespace)
		if err != nil {
			return kubeconfigs, err
		}

		webhooksServiceAccount := newWebhooksServiceAccountMutator(valHelper).Empty()
		kubeconfigs.WebhooksKubeconfig, err = resources.CreateKubeconfig(ctx, values.ResourceCluster, webhooksServiceAccount.Name, webhooksServiceAccount.Namespace)
		if err != nil {
			return kubeconfigs, err
		}

		userServiceAccount := newUserServiceAccountMutator(valHelper).Empty()
		kubeconfigs.UserKubeconfig, err = resources.CreateKubeconfig(ctx, values.ResourceCluster, userServiceAccount.Name, userServiceAccount.Namespace)
		if err != nil {
			return kubeconfigs, err
		}
	}

	return kubeconfigs, nil
}

func UninstallLandscaperRBACResources(ctx context.Context, values *Values) error {

	valHelper, err := newValuesHelper(values)
	if err != nil {
		return err
	}

	resourceClient := values.ResourceCluster.Client()

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
