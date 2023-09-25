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

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/landscaper/apis/config"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/pkg/utils"
)

// DeployInternalDeployers automatically deploys configured deployers using the new Deployer registrations.
func (o *Options) DeployInternalDeployers(ctx context.Context, kubeClient client.Client, config *config.LandscaperConfiguration) error {
	commonCompDescRef := &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: config.DeployerManagement.DeployerRepositoryContext,
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

	apply := func(reg *lsv1alpha1.DeployerRegistration, args DeployerApplyArgs) error {
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
			return fmt.Errorf("unable to create deployer values: %w", err)
		}
		if args.Registration.Spec.InstallationTemplate.ImportDataMappings == nil {
			args.Registration.Spec.InstallationTemplate.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{}
		}
		args.Registration.Spec.InstallationTemplate.ImportDataMappings["values"] = lsv1alpha1.NewAnyJSON(valuesBytes)

		// apply args registrations values to registration
		reg.Spec.InstallationTemplate = args.Registration.Spec.InstallationTemplate
		reg.Spec.DeployItemTypes = args.Registration.Spec.DeployItemTypes
		return nil
	}

	for _, deployerName := range o.EnabledDeployers {
		if err := o.deployDeployerRegistrations(ctx, deployerName, kubeClient, apply); err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) deployDeployerRegistrations(ctx context.Context, deployerName string, kubeClient client.Client, apply DeployerApplyFunc) error {
	log, ctx := logging.FromContextOrNew(ctx, nil, lc.KeyMethod, "deployDeployerRegistrations")
	log.Info("Enable Deployer", lc.KeyResourceNonNamespaced, deployerName)

	deployerConfig := o.GetDeployerConfigForDeployer(deployerName)

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
		reg := deployerArg.Registration.DeepCopy()
		if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, reg, func() error {
			return apply(reg, deployerArg)
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

	reg := deployerArg.Registration.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(ctx, kubeClient, reg, func() error {
		return apply(reg, deployerArg)
	}); err != nil {
		return fmt.Errorf("unable to create Deployer registration for %q: %w", deployerName, err)
	}
	return nil
}
