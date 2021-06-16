// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/utils"
)

// DeployInternalDeployers automatically deploys configured deployers using the new Deployer registrations.
func (o *Options) DeployInternalDeployers(ctx context.Context, log logr.Logger, kubeClient client.Client, config *config.LandscaperConfiguration) error {
	repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("eu.gcr.io/gardener-project/development", ""))
	if err != nil {
		return fmt.Errorf("unable to parse repository context: %w", err)
	}
	commonCompDescRef := &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repoCtx,
			ComponentName:     "",
			Version:           o.Version,
		},
	}

	// read oci credentials and add them to the deployers
	values := make(map[string]interface{})
	if config.Registry.OCI != nil {
		ociAuthConfig := make(map[string]interface{})
		for _, file := range config.Registry.OCI.ConfigFiles {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("unable to read docker auth Config from %q: %w", file, err)
			}
			var auth interface{}
			if err := json.Unmarshal(data, &auth); err != nil {
				return fmt.Errorf("unable to parse oci configuration from %q: %w", file, err)
			}

			ociAuthConfig[path.Base(file)] = auth
		}
		values["deployer"] = map[string]interface{}{
			"oci": map[string]interface{}{
				"allowPlainHttp":     config.Registry.OCI.AllowPlainHttp,
				"insecureSkipVerify": config.Registry.OCI.InsecureSkipVerify,
				"secrets":            ociAuthConfig,
			},
		}
	}

	apply := func(args DeployerApplyArgs) error {
		if args.Registration.Spec.InstallationTemplate.ComponentDescriptor == nil {
			args.Registration.Spec.InstallationTemplate.ComponentDescriptor = commonCompDescRef.DeepCopy()
		}
		if len(args.Registration.Spec.DeployItemTypes) == 0 && len(args.Type) != 0 {
			args.Registration.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{args.Type}
		}
		if len(args.Registration.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName) == 0 && len(args.ComponentName) != 0 {
			args.Registration.Spec.InstallationTemplate.ComponentDescriptor.Reference.ComponentName = args.ComponentName
		}
		if (args.Registration.Spec.InstallationTemplate.Blueprint.Reference == nil ||
			len(args.Registration.Spec.InstallationTemplate.Blueprint.Reference.ResourceName) == 0) &&
			len(args.ResourceName) != 0 {
			args.Registration.Spec.InstallationTemplate.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
				ResourceName: args.ResourceName,
			}
		}

		valuesBytes, err := json.Marshal(utils.MergeMaps(values, args.Values))
		if err != nil {
			return fmt.Errorf("unable to create Deployer values: %w", err)
		}
		if args.Registration.Spec.InstallationTemplate.ImportDataMappings == nil {
			args.Registration.Spec.InstallationTemplate.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{}
		}
		args.Registration.Spec.InstallationTemplate.ImportDataMappings["values"] = lsv1alpha1.NewAnyJSON(valuesBytes)
		return nil
	}

	for _, deployerName := range o.EnabledDeployers {
		if err := o.deployInternalDeployer(ctx, log, deployerName, kubeClient, apply); err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) deployInternalDeployer(ctx context.Context, log logr.Logger, deployerName string, kubeClient client.Client, apply DeployerApplyFunc) error {
	log.Info("Enable Deployer", "name", deployerName)

	deployerConfig, _ := o.GetDeployerConfigForDeployer(deployerName)

	if deployerConfig.IsRegistrationType() {
		deployerArg, ok := DefaultDeployerConfiguration[deployerName]
		if !ok {
			deployerArg = DeployerApplyArgs{}
		}
		deployerArg.Registration = deployerConfig.DeployerRegistration
		if len(deployerArg.Registration.Name) == 0 {
			deployerArg.Registration.Name = deployerName
		}
		if val, ok := deployerArg.Registration.Spec.InstallationTemplate.ImportDataMappings["values"]; ok {
			var values map[string]interface{}
			if err := json.Unmarshal(val.RawMessage, &values); err != nil {
				return fmt.Errorf("unable to parse registration values: %w", err)
			}
			deployerArg.Values = values
		}
		if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, deployerArg.Registration, func() error {
			return apply(deployerArg)
		}); err != nil {
			return fmt.Errorf("unable to create Deployer registration for %q: %w", deployerName, err)
		}
		return nil
	}

	deployerArg, ok := DefaultDeployerConfiguration[deployerName]
	if !ok {
		return fmt.Errorf("unknown default deployer %s", deployerName)
	}
	deployerArg.Registration = &lsv1alpha1.DeployerRegistration{}
	deployerArg.Registration.Name = deployerName
	if deployerConfig.IsValueType() {
		deployerArg.Values = deployerConfig.Values
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, deployerArg.Registration, func() error {
		if err := apply(deployerArg); err != nil {
			return err
		}
		if deployerName == "helm" {
			// create a special target for the helm Deployer to not touch the target that is already handled
			// by the agent integrated helm Deployer.
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
			deployerArg.Registration.Spec.InstallationTemplate.ImportDataMappings["targetSelectors"] = lsv1alpha1.NewAnyJSON(targetSelectorBytes)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("unable to create Deployer registration for %q: %w", deployerName, err)
	}
	return nil
}
