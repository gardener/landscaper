// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"errors"
	goflag "flag"
	"io/ioutil"
	"strings"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/kubernetes"
	"github.com/gardener/landscaper/pkg/logger"
	webhook "github.com/gardener/landscaper/pkg/utils/webhook"
)

// constant
func defaultWebhookedResources() []webhook.WebhookedResourceDefinition {
	return []webhook.WebhookedResourceDefinition{
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
}

type options struct {
	log        logr.Logger
	configPath string
	deployers  string

	config           *config.LandscaperConfiguration
	enabledDeployers []string
	webhook          *webhookOptions
}

// options for the webhook (as given to the CLI)
type rawWebhookOptions struct {
	disabledWebhooks            string // lists disabled webhooks as a comma-separated string
	webhookServiceNamespaceName string // webhook service namespace and name in the format <namespace>/<name>
	webhookServerPort           int    // port of the webhook server
	webhookServicePort          int32  // port of the webhook service
}

// options for the webhook (generated from raw CLI options for easier usage)
type webhookOptions struct {
	webhookServiceNamespace string                                // webhook service namespace
	webhookServiceName      string                                // webhook service name
	webhookServerPort       int                                   // port of the webhook server
	webhookServicePort      int32                                 // port of the webhook service
	enabledWebhooks         []webhook.WebhookedResourceDefinition // which resources should be watched by the webhook
	raw                     *rawWebhookOptions                    // the raw values from which these options were generated
}

func NewOptions() *options {
	return &options{
		webhook: &webhookOptions{
			raw: &rawWebhookOptions{},
		},
	}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	fs.StringVar(&o.deployers, "deployers", "",
		`Specify additional deployers that should be enabled.
Controllers are specified as a comma separated list of controller names.
Available deployers are mock,helm,container.`)
	fs.StringVar(&o.webhook.raw.webhookServiceNamespaceName, "webhook-service", "", "Specify namespace and name of the webhook service (format: <namespace>/<name>)")
	fs.StringVar(&o.webhook.raw.disabledWebhooks, "disable-webhooks", "", "Specify validation webhooks that should be disabled ('all' to disable validation completely)")
	fs.IntVar(&o.webhook.raw.webhookServerPort, "webhook-server-port", 9443, "Specify the port for the webhook server")
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

	if len(o.deployers) != 0 {
		o.enabledDeployers = strings.Split(o.deployers, ",")
	}

	err = o.validate() // validate options
	if err != nil {
		return err
	}
	o.webhook.completeWebhookOptions() // compute easier-to-work-with values from the raw webhook options

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

// completeWebhookOptions populates the fields of the webhookOptions object by evaluating the rawWebhookOptions in it
// this functions assumes that the rawWebhookOptions have been validated
func (wo *webhookOptions) completeWebhookOptions() {
	wo.webhookServerPort = wo.raw.webhookServerPort
	wo.webhookServicePort = wo.raw.webhookServicePort
	webhookService := strings.Split(wo.raw.webhookServiceNamespaceName, "/")
	wo.webhookServiceNamespace = webhookService[0]
	wo.webhookServiceName = webhookService[1]
	wo.enabledWebhooks = filterWebhookedResources(defaultWebhookedResources(), strings.Split(wo.raw.disabledWebhooks, ","))
}

// validateRawWebhookOptions validates the webhook options as given to the CLI
func validateRawWebhookOptions(wo *rawWebhookOptions) field.ErrorList {
	if wo == nil {
		return field.ErrorList{field.InternalError(field.NewPath("rawWebhookOptions"), errors.New("must not be nil"))}
	}
	allErrs := field.ErrorList{}
	disabled := false
	if len(wo.disabledWebhooks) != 0 { // something has been disabled
		allowedWebhooks := []string{} // needed for logging
		for _, elem := range defaultWebhookedResources() {
			allowedWebhooks = append(allowedWebhooks, elem.ResourceName)
		}
		allowedWebhooks = append(allowedWebhooks, "all")
		disabledWebhooks := strings.Split(wo.disabledWebhooks, ",")
		for _, elem := range disabledWebhooks {
			if elem == "all" {
				continue
			}
			// check whether the webhooks to be disabled are actually known
			valid := false
			for _, tmp := range defaultWebhookedResources() {
				if elem == tmp.ResourceName {
					valid = true
					break
				}
			}
			if !valid {
				allErrs = append(allErrs, field.NotSupported(field.NewPath("--disable-webhooks"), elem, allowedWebhooks))
			}
		}
		wr := filterWebhookedResources(defaultWebhookedResources(), disabledWebhooks) // compute list of enabled webhooks to decide if webhooks are completely disabled
		disabled = (len(wr) == 0)                                                     // true if all webhooks have been disabled, either by 'all' or by listing all possible ones in the '--disable-webhooks' option
	}
	// validate service name and namespace
	if !disabled { // service namespace not needed if completely disabled
		if len(wo.webhookServiceNamespaceName) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("--webhook-service"), "option is required unless all webhooks are disabled"))
		} else {
			ws := strings.Split(wo.webhookServiceNamespaceName, "/")
			if len(ws) < 2 {
				allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-service"), wo.webhookServiceNamespaceName, "must have the format '<namespace>/<name>'"))
			}
		}
	}
	// validate ports
	if wo.webhookServerPort < 0 { // is defaulted on 0 value
		allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-server-port"), wo.webhookServerPort, "must not be below zero"))
	}
	if wo.webhookServicePort <= 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("--webhook-service-port"), wo.webhookServicePort, "must be greater than zero"))
	}
	return allErrs
}

// filterWebhookedResources returns a slice of WebhookedResourceDefinitions that contains only those of the given webhookedResources whose ResourceName is not specified in disabledWebhooks
func filterWebhookedResources(webhookedResources []webhook.WebhookedResourceDefinition, disabledWebhooks []string) []webhook.WebhookedResourceDefinition {
	fwr := []webhook.WebhookedResourceDefinition{}
	for _, dw := range disabledWebhooks {
		if dw == "all" {
			return fwr // return empty slice if all webhooks have been disabled
		}
	}
	for _, wr := range webhookedResources {
		enabled := true
		for _, dw := range disabledWebhooks {
			if wr.ResourceName == dw {
				enabled = false
				break
			}
		}
		if enabled {
			fwr = append(fwr, wr)
		}
	}
	return fwr
}
