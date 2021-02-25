// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	install "github.com/gardener/landscaper/apis/core/install"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	executionactuator "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	installationsactuator "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/version"

	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
	webhook "github.com/gardener/landscaper/pkg/utils/webhook"
)

func NewLandscaperControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-controller",
		Short: "Landscaper controller manages the orchestration of components",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.log.Error(err, "unable to run landscaper controller")
				os.Exit(1)
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.log.Info(fmt.Sprintf("Start Landscaper Controller with version %q", version.Get().String()))

	opts := manager.Options{
		LeaderElection:     false,
		Port:               9443,
		MetricsBindAddress: "0",
	}

	if o.config.Metrics != nil {
		opts.MetricsBindAddress = fmt.Sprintf(":%d", o.config.Metrics.Port)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	componentcliMetrics.RegisterCacheMetrics(controllerruntimeMetrics.Registry)

	crdmgr, err := crdmanager.NewCrdManager(ctrl.Log.WithName("setup").WithName("CRDManager"), mgr, o.config)
	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	install.Install(mgr.GetScheme())

	if err := installationsactuator.AddActuatorToManager(mgr, o.config); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactuator.AddActuatorToManager(mgr); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	// create ValidatingWebhookConfiguration and register webhooks, if validation is enabled, delete it otherwise
	if err := registerWebhooks(ctx, mgr, o); err != nil {
		return fmt.Errorf("unable to register validation webhook: %w", err)
	}

	for _, deployerName := range o.deployer.EnabledDeployers {
		o.log.Info("Enable Deployer", "name", deployerName)
		if deployerName == "container" {
			config := &containerv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			containerctlr.DefaultConfiguration(config)
			if err := containerctlr.AddActuatorToManager(mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add container deployer: %w", err)
			}
		} else if deployerName == "helm" {
			config := &helmv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			if err := helmctlr.AddActuatorToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "manifest" {
			config := &manifest.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, config); err != nil {
				return err
			}
			if err := manifestctlr.AddActuatorToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "mock" {
			if err := mockctlr.AddActuatorToManager(mgr); err != nil {
				return fmt.Errorf("unable to add mock deployer: %w", err)
			}
		} else {
			return fmt.Errorf("unknown deployer %s", deployerName)
		}
	}

	o.log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}

func registerWebhooks(ctx context.Context, mgr manager.Manager, o *options) error {
	webhookLogger := ctrl.Log.WithName("webhook").WithName("validation")
	webhookConfigurationName := "landscaper-validation-webhook"
	// noop if all webhooks are disabled
	if len(o.webhook.enabledWebhooks) == 0 {
		webhookLogger.Info("Validation disabled")
		return webhook.DeleteValidatingWebhookConfiguration(ctx, mgr, webhookConfigurationName, webhookLogger)
	}

	webhookLogger.Info("Validation enabled")

	// initialize webhook options
	wo := webhook.Options{
		WebhookConfigurationName: webhookConfigurationName,
		WebhookBasePath:          "/webhook/validate/",
		WebhookNameSuffix:        ".validation.landscaper.gardener.cloud",
		ObjectSelector: metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Operator: metav1.LabelSelectorOpNotIn,
					Key:      "validation.landscaper.gardener.cloud/skip-validation",
					Values:   []string{"true"},
				},
			},
		},
		ServicePort:        o.webhook.webhookServicePort,
		ServiceName:        o.webhook.webhookServiceName,
		ServiceNamespace:   o.webhook.webhookServiceNamespace,
		WebhookedResources: o.webhook.enabledWebhooks,
	}

	// generate certificates
	mgr.GetWebhookServer().CertDir = filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	var err error
	wo.CABundle, err = webhook.GenerateCertificates(ctx, mgr, mgr.GetWebhookServer().CertDir, wo.ServiceNamespace, wo.ServiceName)
	if err != nil {
		return fmt.Errorf("unable to generate webhook certificates: %w", err)
	}

	// log which resources are being watched
	webhookedResourcesLog := []string{}
	for _, elem := range wo.WebhookedResources {
		webhookedResourcesLog = append(webhookedResourcesLog, elem.ResourceName)
	}
	webhookLogger.Info("Enabling validation", "resources", webhookedResourcesLog)

	// create/update/delete ValidatingWebhookConfiguration
	if err := webhook.UpdateValidatingWebhookConfiguration(ctx, mgr, wo, webhookLogger); err != nil {
		return err
	}
	// register webhooks
	if err := webhook.RegisterWebhooks(ctx, mgr, wo, webhookLogger); err != nil {
		return err
	}

	return nil
}
