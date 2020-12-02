// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	install "github.com/gardener/landscaper/apis/core/install"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	manifestv1alpha1 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha1"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	executionactuator "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	installationsactuator "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"

	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	webhook "github.com/gardener/landscaper/pkg/utils/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
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

	opts := manager.Options{
		LeaderElection:     false,
		Port:               o.webhookServerPort,
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

	install.Install(mgr.GetScheme())

	if err := installationsactuator.AddActuatorToManager(mgr, o.config); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactuator.AddActuatorToManager(mgr); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	///// WEBHOOK /////

	wo := webhook.Options{
		WebhookConfigurationName: "landscaper-validation-webhook",
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
		ServicePort: o.webhookServicePort,
	}
	webhookLogger := ctrl.Log.WithName("webhook").WithName("validation")
	webhookedResources := []webhook.WebhookedResourceDefinition{}
	// adds a validation webhook for Installations and Executions
	if o.disabledWebhooks == nil || !o.disabledWebhooks["all"] {
		if o.webhookNamespace == "" {
			return errors.New("webhook service namespace must not be empty")
		}
		if o.webhookName == "" {
			return errors.New("webhook service name must not be empty")
		}
		wo.ServiceNamespace = o.webhookNamespace
		wo.ServiceName = o.webhookName
		// determine resources watched by webhook
		webhookedResourcesTemplate := []webhook.WebhookedResourceDefinition{
			{
				APIGroup:     "landscaper.gardener.cloud",
				APIVersions:  []string{"v1alpha1"},
				ResourceName: "installations",
			},
			{
				APIGroup:     "landscaper.gardener.cloud",
				APIVersions:  []string{"v1alpha1"},
				ResourceName: "deployitems",
			},
			{
				APIGroup:     "landscaper.gardener.cloud",
				APIVersions:  []string{"v1alpha1"},
				ResourceName: "executions",
			},
		}
		webhookedResourcesLog := []string{}
		for _, elem := range webhookedResourcesTemplate {
			if !o.disabledWebhooks[elem.ResourceName] {
				webhookedResources = append(webhookedResources, elem)
				webhookedResourcesLog = append(webhookedResourcesLog, elem.ResourceName)
			}
		}
		wo.WebhookedResources = webhookedResources
		webhookLogger.Info("Enabling validation", "resources", webhookedResourcesLog)
	}

	if err := webhook.ApplyValidatingWebhookConfiguration(ctx, mgr, wo, webhookLogger); err != nil {
		return err
	}

	///// WEBHOOK END /////

	for _, deployerName := range o.enabledDeployers {
		if deployerName == "container" {
			config := &containerv1alpha1.Configuration{
				OCI: o.config.Registry.OCI,
			}
			containerctlr.DefaultConfiguration(config)
			if err := containerctlr.AddActuatorToManager(mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add container deployer: %w", err)
			}
		} else if deployerName == "helm" {
			config := &helmv1alpha1.Configuration{
				OCI: o.config.Registry.OCI,
			}
			if err := helmctlr.AddActuatorToManager(mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "manifest" {
			config := &manifestv1alpha1.Configuration{}
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

	o.log.Info("starting the controller")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}
