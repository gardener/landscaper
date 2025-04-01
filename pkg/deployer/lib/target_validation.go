package lib

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

func ValidateTarget(ctx context.Context, resolvedTarget *lsv1alpha1.ResolvedTarget) error {
	if resolvedTarget == nil {
		// Nothing to validate. Here we do not judge whether a target is required or not.
		return nil
	}

	if resolvedTarget.Target == nil {
		return fmt.Errorf("resolved target does not contain original target")
	}

	targetConfig := &targettypes.KubernetesClusterTargetConfig{}
	if err := yaml.Unmarshal([]byte(resolvedTarget.Content), targetConfig); err != nil {
		return fmt.Errorf("unable to parse target confíguration: %w", err)
	}

	if targetConfig.Kubeconfig.StrVal != nil {
		kubeconfigBytes := []byte(*targetConfig.Kubeconfig.StrVal)
		return validateKubeconfig(ctx, kubeconfigBytes)
	}

	return nil
}

func validateKubeconfig(ctx context.Context, kubeconfigBytes []byte) error {
	logger, _ := logging.FromContextOrNew(ctx, nil)

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return err
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return err
	}

	validFields := []string{"client-certificate-data", "client-key-data", "token", "username", "password"}

	for user, authInfo := range rawConfig.AuthInfos {
		switch {
		case authInfo.ClientCertificate != "":
			logger.Info("client certificate files are not supported in a target kubeconfig")
			return fmt.Errorf("client certificate files are not supported in a target kubeconfig (user %q), the valid fields are: %+v", user, validFields)
		case authInfo.ClientKey != "":
			logger.Info("client key files are not supported in a target kubeconfig")
			return fmt.Errorf("client key files are not supported in a target kubeconfig (user %q), the valid fields are: %+v", user, validFields)
		case authInfo.TokenFile != "":
			logger.Info("token files are not supported in a target kubeconfig")
			return fmt.Errorf("token files are not supported in a target kubeconfig (user %q), the valid fields are: %+v", user, validFields)
		case authInfo.Impersonate != "" || len(authInfo.ImpersonateGroups) > 0:
			logger.Info("impersonation is not supported in a target kubeconfig")
			return fmt.Errorf("impersonation is not supported in a target kubeconfig, the valid fields are: %+v", validFields)
		case authInfo.AuthProvider != nil && len(authInfo.AuthProvider.Config) > 0:
			logger.Info("auth provider configurations are not supported in a target kubeconfig")
			return fmt.Errorf("auth provider configurations are not supported in a target kubeconfig (user %q), the valid fields are: %+v", user, validFields)
		case authInfo.Exec != nil:
			logger.Info("exec configurations are not supported in a target kubeconfig")
			return fmt.Errorf("exec configurations are not supported in a target kubeconfig (user %q), the valid fields are: %+v", user, validFields)
		}
	}
	return nil
}
