// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	coctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/componentoverwrites"
	"github.com/gardener/landscaper/pkg/landscaper/registry/componentoverwrites"

	"github.com/gardener/landscaper/pkg/agent"

	"github.com/gardener/landscaper/pkg/landscaper/controllers/deployers"

	install "github.com/gardener/landscaper/apis/core/install"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	helmv1alpha1 "github.com/gardener/landscaper/apis/deployer/helm/v1alpha1"
	manifestv1alpha2 "github.com/gardener/landscaper/apis/deployer/manifest/v1alpha2"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	containerctlr "github.com/gardener/landscaper/pkg/deployer/container"
	helmctlr "github.com/gardener/landscaper/pkg/deployer/helm"
	manifestctlr "github.com/gardener/landscaper/pkg/deployer/manifest"
	mockctlr "github.com/gardener/landscaper/pkg/deployer/mock"
	deployitemctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/deployitem"
	executionactrl "github.com/gardener/landscaper/pkg/landscaper/controllers/execution"
	installationsctrl "github.com/gardener/landscaper/pkg/landscaper/controllers/installations"
	"github.com/gardener/landscaper/pkg/version"

	componentcliMetrics "github.com/gardener/component-cli/ociclient/metrics"
	controllerruntimeMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/gardener/landscaper/pkg/landscaper/crdmanager"
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

	if err := crdmgr.EnsureCRDs(ctx); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	install.Install(mgr.GetScheme())

	ctrlLogger := o.log.WithName("controllers")
	componentOverwriteMgr := componentoverwrites.New()
	if err := coctrl.AddControllerToManager(ctrlLogger, mgr, componentOverwriteMgr); err != nil {
		return fmt.Errorf("unable to setup commponent overwrites controller: %w", err)
	}

	if err := installationsctrl.AddControllerToManager(ctrlLogger, mgr, componentOverwriteMgr, o.config); err != nil {
		return fmt.Errorf("unable to setup installation controller: %w", err)
	}

	if err := executionactrl.AddControllerToManager(ctrlLogger, mgr); err != nil {
		return fmt.Errorf("unable to setup execution controller: %w", err)
	}

	if err := deployitemctrl.AddControllerToManager(ctrlLogger, mgr, o.config.DeployItemTimeouts.Pickup, o.config.DeployItemTimeouts.Abort, o.config.DeployItemTimeouts.ProgressingDefault); err != nil {
		return fmt.Errorf("unable to setup deployitem controller: %w", err)
	}

	if !o.config.DeployerManagement.Disable {
		if err := deployers.AddControllersToManager(ctrlLogger, mgr, o.config); err != nil {
			return fmt.Errorf("unable to setup deployer controllers: %w", err)
		}
		if !o.config.DeployerManagement.Agent.Disable {
			agentConfig := o.config.DeployerManagement.Agent.AgentConfiguration
			// add default selector and in addition reconcile all target that do not have a a environment definition
			agentConfig.TargetSelectors = append(agent.DefaultTargetSelector(agentConfig.Name), lsv1alpha1.TargetSelector{
				Annotations: []lsv1alpha1.Requirement{
					{
						Key:      lsv1alpha1.DeployerEnvironmentTargetAnnotationName,
						Operator: selection.DoesNotExist,
					},
				},
			})
			if err := agent.AddToManager(ctx, o.log, mgr, mgr, agentConfig); err != nil {
				return fmt.Errorf("unable to setup default agent: %w", err)
			}
		}
		if err := o.deployInternalDeployers(ctx, mgr); err != nil {
			return err
		}
	} else {
		if err := o.deployLegacyInternalDeployers(mgr); err != nil {
			return err
		}
	}

	o.log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}

// deployInternalDeployers automatically deploys configured deployers using the new deployer registrations.
func (o *options) deployInternalDeployers(ctx context.Context, mgr manager.Manager) error {
	directClient, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return fmt.Errorf("unable to create direct client: %q", err)
	}
	commonCompDescRef := &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &cdv2.RepositoryContext{
				Type:    cdv2.OCIRegistryType,
				BaseURL: "eu.gcr.io/gardener-project/development",
			},
			ComponentName: "",
			Version:       o.deployer.Version,
		},
	}

	// read oci credentials and add them to the deployers
	values := make(map[string]interface{})
	if o.config.Registry.OCI != nil {
		ociAuthConfig := make(map[string]interface{})
		for _, file := range o.config.Registry.OCI.ConfigFiles {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("unable to read docker auth config from %q: %w", file, err)
			}
			var auth interface{}
			if err := json.Unmarshal(data, &auth); err != nil {
				return fmt.Errorf("unable to parse oci configuration from %q: %w", file, err)
			}

			ociAuthConfig[path.Base(file)] = auth
		}
		values["deployer"] = map[string]interface{}{
			"oci": map[string]interface{}{
				"allowPlainHttp":     o.config.Registry.OCI.AllowPlainHttp,
				"insecureSkipVerify": o.config.Registry.OCI.InsecureSkipVerify,
				"secrets":            ociAuthConfig,
			},
		}
	}

	valuesBytes, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("unable to create deployer values: %w", err)
	}

	apply := func(reg *lsv1alpha1.DeployerRegistration, dtype lsv1alpha1.DeployItemType, componentName, resourceName string) {
		reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{dtype}
		reg.Spec.InstallationTemplate.ComponentDescriptor = commonCompDescRef
		reg.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName = componentName
		reg.Spec.InstallationTemplate.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resourceName,
		}
		reg.Spec.InstallationTemplate.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
			"values": lsv1alpha1.NewAnyJSON(valuesBytes),
		}
	}
	for _, deployerName := range o.deployer.EnabledDeployers {
		o.log.Info("Enable Deployer", "name", deployerName)
		if deployerName == "container" {
			reg := &lsv1alpha1.DeployerRegistration{}
			reg.Name = deployerName

			if _, err := controllerutil.CreateOrUpdate(ctx, directClient, reg, func() error {
				apply(reg,
					containerctlr.Type,
					"github.com/gardener/landscaper/container-deployer",
					"container-deployer-blueprint")
				return nil
			}); err != nil {
				return fmt.Errorf("unable to create deployer registration for %q: %w", deployerName, err)
			}
		} else if deployerName == "helm" {
			reg := &lsv1alpha1.DeployerRegistration{}
			reg.Name = deployerName

			// create a special target for the helm deployer to not touch the target that is already handled
			// by the agent integrated helm deployer.
			targetSelectorBytes, err := json.Marshal([]lsv1alpha1.TargetSelector{
				{
					Annotations: []lsv1alpha1.Requirement{
						{
							Key:      lsv1alpha1.DeployerEnvironmentTargetAnnotationName,
							Operator: selection.DoesNotExist,
						},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("unable to marshal helm target selector: %w", err)
			}
			if _, err := controllerutil.CreateOrUpdate(ctx, directClient, reg, func() error {
				apply(reg,
					helmctlr.Type,
					"github.com/gardener/landscaper/helm-deployer",
					"helm-deployer-blueprint")
				reg.Spec.InstallationTemplate.ImportDataMappings["targetSelectors"] = lsv1alpha1.NewAnyJSON(targetSelectorBytes)
				return nil
			}); err != nil {
				return fmt.Errorf("unable to create deployer registration for %q: %w", deployerName, err)
			}
		} else if deployerName == "manifest" {
			reg := &lsv1alpha1.DeployerRegistration{}
			reg.Name = deployerName

			if _, err := controllerutil.CreateOrUpdate(ctx, directClient, reg, func() error {
				apply(reg,
					manifestctlr.Type,
					"github.com/gardener/landscaper/manifest-deployer",
					"manifest-deployer-blueprint")

				return nil
			}); err != nil {
				return fmt.Errorf("unable to create deployer registration for %q: %w", deployerName, err)
			}
		} else if deployerName == "mock" {
			reg := &lsv1alpha1.DeployerRegistration{}
			reg.Name = deployerName

			if _, err := controllerutil.CreateOrUpdate(ctx, directClient, reg, func() error {
				apply(reg,
					mockctlr.Type,
					"github.com/gardener/landscaper/mock-deployer",
					"mock-deployer-blueprint")
				return nil
			}); err != nil {
				return fmt.Errorf("unable to create deployer registration for %q: %w", deployerName, err)
			}
		} else {
			return fmt.Errorf("unknown deployer %s", deployerName)
		}
	}
	return nil
}

// deployLegacyInternalDeployers includes the deployers directly in the manager of the landscaper.
// This method is deprecated and the newer deployer registrations should be used.
func (o *options) deployLegacyInternalDeployers(mgr manager.Manager) error {
	log := o.log.WithName("inlineDeployer")
	for _, deployerName := range o.deployer.EnabledDeployers {
		o.log.Info("Enable Deployer", "name", deployerName)
		if deployerName == "container" {
			config := containerv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, &config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			containerctlr.DefaultConfiguration(&config)
			if err := containerctlr.AddControllerToManager(log, mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add container deployer: %w", err)
			}
		} else if deployerName == "helm" {
			config := helmv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, &config); err != nil {
				return err
			}
			config.OCI = o.config.Registry.OCI
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := helmctlr.AddDeployerToManager(log, mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "manifest" {
			config := manifestv1alpha2.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, &config); err != nil {
				return err
			}
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := manifestctlr.AddDeployerToManager(log, mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add helm deployer: %w", err)
			}
		} else if deployerName == "mock" {
			config := mockv1alpha1.Configuration{}
			if err := o.deployer.GetDeployerConfiguration(deployerName, &config); err != nil {
				return err
			}
			config.TargetSelector = addDefaultTargetSelector(config.TargetSelector)
			if err := mockctlr.AddDeployerToManager(log, mgr, mgr, config); err != nil {
				return fmt.Errorf("unable to add mock deployer: %w", err)
			}
		} else {
			return fmt.Errorf("unknown deployer %s", deployerName)
		}
	}
	return nil
}
