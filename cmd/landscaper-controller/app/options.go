// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"io/ioutil"
	"strings"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/apis/config"
	containerinstall "github.com/gardener/landscaper/apis/deployer/container/install"
	helminstall "github.com/gardener/landscaper/apis/deployer/helm/install"
	manifestinstall "github.com/gardener/landscaper/apis/deployer/manifest/install"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/logger"
	webhook "github.com/gardener/landscaper/pkg/utils/webhook"
)

// constant
func defaultWebhookedResources() map[string]webhook.WebhookedResourceDefinition {
	return map[string]webhook.WebhookedResourceDefinition{
		"installations": {
			APIGroup:     "landscaper.gardener.cloud",
			APIVersions:  []string{"v1alpha1"},
			ResourceName: "installations",
		},
		"deployitems": {
			APIGroup:     "landscaper.gardener.cloud",
			APIVersions:  []string{"v1alpha1"},
			ResourceName: "deployitems",
		},
		"executions": {
			APIGroup:     "landscaper.gardener.cloud",
			APIVersions:  []string{"v1alpha1"},
			ResourceName: "executions",
		},
	}
}

type options struct {
	log        logr.Logger
	configPath string

	config   *config.LandscaperConfiguration
	deployer deployerOptions
	webhook  webhookOptions
}

func NewOptions() *options {
	return &options{
		webhook: webhookOptions{
			raw: rawWebhookOptions{},
		},
	}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.deployer.deployers, "deployers", "",
		`Specify additional deployers that should be enabled.
Controllers are specified as a comma separated list of controller names.
Available deployers are mock,helm,container.`)
	fs.StringVar(&o.deployer.deployersConfigPath, "deployers-config", "", "Specify the path to the deployers-configuration file")
	fs.StringVar(&o.webhook.raw.webhookServiceNamespaceName, "webhook-service", "", "Specify namespace and name of the webhook service (format: <namespace>/<name>)")
	fs.StringVar(&o.webhook.raw.disabledWebhooks, "disable-webhooks", "", "Specify validation webhooks that should be disabled ('all' to disable validation completely)")
	fs.Int32Var(&o.webhook.raw.webhookServicePort, "webhook-service-port", 9443, "Specify the port of the webhook service")
	logger.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	o.config, err = o.parseConfigurationFile()
	if err != nil {
		return err
	}
	if err := o.deployer.Complete(); err != nil {
		return err
	}

	err = o.validate() // validate options
	if err != nil {
		return err
	}
	err = o.webhook.completeWebhookOptions() // compute easier-to-work-with values from the raw webhook options
	if err != nil {
		return err
	}

	return nil
}

func (o *options) parseConfigurationFile() (*config.LandscaperConfiguration, error) {
	decoder := serializer.NewCodecFactory(kubernetes.ConfigScheme).UniversalDecoder()
	if len(o.configPath) == 0 {
		return &config.LandscaperConfiguration{}, nil
	}
	data, err := ioutil.ReadFile(o.configPath)
	if err != nil {
		return nil, err
	}

	cfg := &config.LandscaperConfiguration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validates the options
func (o *options) validate() error {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateRawWebhookOptions(o.webhook.raw)...)

	if len(allErrs) > 0 {
		return allErrs.ToAggregate()
	}
	return nil
}

/////////////////////
// Webhook Options //
/////////////////////

// options for the webhook (as given to the CLI)
type rawWebhookOptions struct {
	disabledWebhooks            string // lists disabled webhooks as a comma-separated string
	webhookServiceNamespaceName string // webhook service namespace and name in the format <namespace>/<name>
	webhookServicePort          int32  // port of the webhook service
}

// options for the webhook (generated from raw CLI options for easier usage)
type webhookOptions struct {
	webhookServiceNamespace string                                // webhook service namespace
	webhookServiceName      string                                // webhook service name
	webhookServicePort      int32                                 // port of the webhook service
	enabledWebhooks         []webhook.WebhookedResourceDefinition // which resources should be watched by the webhook
	raw                     rawWebhookOptions                     // the raw values from which these options were generated
}

// completeWebhookOptions populates the fields of the webhookOptions object by evaluating the rawWebhookOptions in it
// this functions assumes that the rawWebhookOptions have been validated
func (wo *webhookOptions) completeWebhookOptions() error {
	allErrs := field.ErrorList{}
	wo.webhookServicePort = wo.raw.webhookServicePort
	wo.enabledWebhooks = filterWebhookedResources(defaultWebhookedResources(), stringListToMap(wo.raw.disabledWebhooks))
	if len(wo.enabledWebhooks) > 0 {
		if len(wo.raw.webhookServiceNamespaceName) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("--webhook-service"), "option is required unless all webhooks are disabled"))
		} else {
			webhookService := strings.Split(wo.raw.webhookServiceNamespaceName, "/")
			wo.webhookServiceNamespace = webhookService[0]
			wo.webhookServiceName = webhookService[1]
		}
	}
	if len(allErrs) > 0 {
		return allErrs.ToAggregate()
	}
	return nil
}

// validateRawWebhookOptions validates the webhook options as given to the CLI
func validateRawWebhookOptions(wo rawWebhookOptions) field.ErrorList {
	allErrs := field.ErrorList{}
	dwr := defaultWebhookedResources()
	if len(wo.disabledWebhooks) != 0 { // something has been disabled
		// validate that no unknown values are in the list of to-be-disabled webhooks
		allowedWebhooks := allowedWebhookDisables()
		disabledWebhooks := strings.Split(wo.disabledWebhooks, ",")
		for _, elem := range disabledWebhooks {
			if _, ok := dwr[elem]; (elem != "all") && !ok {
				allErrs = append(allErrs, field.NotSupported(field.NewPath("--disable-webhooks"), elem, allowedWebhooks))
			}
		}
	}
	// validate service name and namespace
	if len(wo.webhookServiceNamespaceName) > 0 {
		ws := strings.Split(wo.webhookServiceNamespaceName, "/")
		if len(ws) < 2 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-service"), wo.webhookServiceNamespaceName, "must have the format '<namespace>/<name>'"))
		} else {
			if len(ws[0]) == 0 || len(ws[1]) == 0 {
				allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-service"), wo.webhookServiceNamespaceName, "neither name nor namespace of the webhook service must be empty"))
			}
		}
	}
	// validate port
	if wo.webhookServicePort <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-service-port"), wo.webhookServicePort, "must be greater than zero"))
	}
	return allErrs
}

// filterWebhookedResources returns a slice of WebhookedResourceDefinitions that contains only those of the given webhookedResources whose ResourceName is not specified in disabledWebhooks
func filterWebhookedResources(webhookedResources map[string]webhook.WebhookedResourceDefinition, disabledWebhooks map[string]bool) []webhook.WebhookedResourceDefinition {
	fwr := []webhook.WebhookedResourceDefinition{}
	if _, ok := disabledWebhooks["all"]; ok {
		return fwr // all webhooks disabled, return empty slice
	}
	for _, wr := range webhookedResources {
		if _, ok := disabledWebhooks[wr.ResourceName]; !ok {
			fwr = append(fwr, wr)
		}
	}
	return fwr
}

// allowedWebhookDisables computes a list of allowed values for the '--disable-webhooks' option
func allowedWebhookDisables() []string {
	dwr := defaultWebhookedResources()
	res := make([]string, len(dwr)+1)
	c := 0
	for _, elem := range dwr {
		res[c] = elem.ResourceName
		c++
	}
	res[c] = "all"
	return res
}

//////////////////////
// Deployer Options //
//////////////////////

type deployerOptions struct {
	deployers           string
	deployersConfigPath string

	EnabledDeployers []string
	DeployersConfig  DeployersConfiguration
}

func (o *deployerOptions) GetDeployerConfiguration(name string, config runtime.Object) error {
	if o.DeployersConfig.Deployers == nil {
		return nil
	}
	data, ok := o.DeployersConfig.Deployers[name]
	if !ok || data.Raw == nil {
		return nil
	}
	deployerScheme := runtime.NewScheme()
	helminstall.Install(deployerScheme)
	manifestinstall.Install(deployerScheme)
	containerinstall.Install(deployerScheme)

	if _, _, err := serializer.NewCodecFactory(deployerScheme).UniversalDecoder().Decode(data.Raw, nil, config); err != nil {
		return err
	}
	return nil
}

func (o *deployerOptions) Complete() error {
	if len(o.deployers) != 0 {
		o.EnabledDeployers = strings.Split(o.deployers, ",")
	}

	if err := o.parseDeployersConfigurationFile(); err != nil {
		return err
	}

	return nil
}

func (o *deployerOptions) parseDeployersConfigurationFile() error {
	if len(o.deployersConfigPath) == 0 {
		return nil
	}
	data, err := ioutil.ReadFile(o.deployersConfigPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, o.DeployersConfig)
}

// stringListToMap turns a comma-separated list of strings into pseudo-set that maps all elements of the list to true
func stringListToMap(opt string) map[string]bool {
	res := map[string]bool{}
	tmp := strings.Split(opt, ",")
	for _, t := range tmp {
		res[t] = true
	}
	return res
}
