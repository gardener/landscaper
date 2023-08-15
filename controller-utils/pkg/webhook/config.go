// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/url"
	"path"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/controller-utils/pkg/utils"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigOptions contains the configuration that is necessary to create a ValidatingWebhookConfiguration or MutatingWebhookConfiguration.
type ConfigOptions struct {
	// Name of the WebhookConfiguration that will be created.
	WebhookConfigurationName string
	// WebhookNameSuffix will be appended to the webhooks' names.
	WebhookNameSuffix string
	// ServiceConfig is the configuration for reaching a webhook exposed via a service (running in the same cluster).
	// It is mutually exclusive with WebhookURL.
	Service *WebhookServiceOptions
	// WebhookURL is used for reaching a webhook running outside of the cluster.
	// It is mutually exclusive with Service.
	WebhookURL string
	// CABundle contains the certificates for the webhook.
	CABundle []byte
}

// WebhookServiceOptions contains the configuration for reaching a webhook which is running in the same cluster and exposed via a service.
// If the webhook server is running in a different cluster, WebhookURL must be used instead.
type WebhookServiceOptions struct {
	// Name is the name of the service under which the webhook can be reached.
	Name string
	// Namespace is the namespace of the webhook service.
	Namespace string
	// Port is the port of the webhook service.
	Port int32
}

// UpdateWebhookConfiguration will create or update a ValidatingWebhookConfiguration or MutatingWebhookConfiguration.
func UpdateWebhookConfiguration(ctx context.Context, wt WebhookType, kubeClient client.Client, wr WebhookRegistry, o *ConfigOptions) error {
	log := logging.FromContextOrDiscard(ctx).WithValues(lc.KeyResourceKind, fmt.Sprintf("%sConfiguration", string(wt)))

	// exactly one of service and webhook url must be set
	if (o.Service == nil) == (o.WebhookURL == "") {
		return fmt.Errorf("invalid webhook configuration options: exactly one of [Service, WebhookURL] must be set")
	}

	var whc client.Object
	var whcUpdate func(client.Object) func() error
	var err error
	switch wt {
	case ValidatingWebhook:
		whc, whcUpdate, err = o.buildValidatingWebhookConfiguration(wr)
	case MutatingWebhook:
		whc, whcUpdate, err = o.buildMutatingWebhookConfiguration(wr)
	}
	if err != nil {
		return fmt.Errorf("error building %s: %w", string(wt), err)
	}

	log.Info("Creating/updating webhook configuration", lc.KeyResource, o.WebhookConfigurationName)
	_, err = ctrl.CreateOrUpdate(ctx, kubeClient, whc, whcUpdate(whc))
	if err != nil {
		return fmt.Errorf("unable to create/update webhook configuration: %w", err)
	}
	log.Info("Webhook configuration created/updated", lc.KeyResource, o.WebhookConfigurationName)

	return nil
}

// DeleteValidatingWebhookConfiguration deletes a ValidatingWebhookConfiguration or MutatingWebhookConfiguration.
func DeleteWebhookConfiguration(ctx context.Context, wt WebhookType, kubeClient client.Client, name string) error {
	log := logging.FromContextOrDiscard(ctx).WithValues(lc.KeyResourceKind, fmt.Sprintf("%sConfiguration", string(wt)))

	var whc client.Object
	switch wt {
	case ValidatingWebhook:
		whc = &admissionregistrationv1.ValidatingWebhookConfiguration{}
	case MutatingWebhook:
		whc = &admissionregistrationv1.MutatingWebhookConfiguration{}
	}
	whc.SetName(name)

	log.Info("Removing webhook configuration, if it exists", lc.KeyResource, name)
	if err := kubeClient.Delete(ctx, whc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Webhook configuration not found", lc.KeyResource, name)
		} else {
			return fmt.Errorf("unable to delete webhook configuration %q: %w", name, err)
		}
	} else {
		log.Info("Webhook configuration deleted", lc.KeyResource, name)
	}
	return nil
}

// commonWebhookConfig contains the fields which ValidatingWebhookConfigurations and MutatingWebhookConfigurations have in common.
type commonWebhookConfig struct {
	Name                    string
	SideEffects             *admissionregistrationv1.SideEffectClass
	FailurePolicy           *admissionregistrationv1.FailurePolicyType
	ObjectSelector          *metav1.LabelSelector
	AdmissionReviewVersions []string
	Rules                   []admissionregistrationv1.RuleWithOperations
	ClientConfig            admissionregistrationv1.WebhookClientConfig
	TimeoutSeconds          *int32
}

func (c *commonWebhookConfig) toValidatingWebhook() admissionregistrationv1.ValidatingWebhook {
	return admissionregistrationv1.ValidatingWebhook{
		Name:                    c.Name,
		SideEffects:             c.SideEffects,
		FailurePolicy:           c.FailurePolicy,
		ObjectSelector:          c.ObjectSelector,
		AdmissionReviewVersions: c.AdmissionReviewVersions,
		Rules:                   c.Rules,
		ClientConfig:            c.ClientConfig,
		TimeoutSeconds:          c.TimeoutSeconds,
	}
}

func (c *commonWebhookConfig) toMutatingWebhook() admissionregistrationv1.MutatingWebhook {
	return admissionregistrationv1.MutatingWebhook{
		Name:                    c.Name,
		SideEffects:             c.SideEffects,
		FailurePolicy:           c.FailurePolicy,
		ObjectSelector:          c.ObjectSelector,
		AdmissionReviewVersions: c.AdmissionReviewVersions,
		Rules:                   c.Rules,
		ClientConfig:            c.ClientConfig,
		TimeoutSeconds:          c.TimeoutSeconds,
	}
}

func (o *ConfigOptions) buildValidatingWebhookConfiguration(wr WebhookRegistry) (*admissionregistrationv1.ValidatingWebhookConfiguration, func(client.Object) func() error, error) {
	vwc := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	vwc.SetName(o.WebhookConfigurationName)

	// construct ValidatingWebhookConfiguration
	vwcWebhooks := []admissionregistrationv1.ValidatingWebhook{}

	for _, w := range wr.Filter(ValidatingWebhooksFilter()) {
		if !w.isInitialized {
			return nil, nil, fmt.Errorf("webhook '%s' is not initialized, please call Initialize on the webhook or InitializeAll on the registry before generating the ValidatingWebhookConfiguration", w.Name)
		}
		whCfg, err := o.buildCommonWebhookConfiguration(w)
		if err != nil {
			return nil, nil, err
		}
		vwcWebhooks = append(vwcWebhooks, whCfg.toValidatingWebhook())
	}

	vwc.Webhooks = vwcWebhooks

	return vwc, func(obj client.Object) func() error {
		return func() error {
			typedObj, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration)
			if !ok {
				return fmt.Errorf("update function for ValidatingWebhookConfigurations called on invalid object")
			}
			typedObj.Webhooks = vwcWebhooks
			return nil
		}
	}, nil
}

func (o *ConfigOptions) buildMutatingWebhookConfiguration(wr WebhookRegistry) (*admissionregistrationv1.MutatingWebhookConfiguration, func(client.Object) func() error, error) {
	mwc := &admissionregistrationv1.MutatingWebhookConfiguration{}
	mwc.SetName(o.WebhookConfigurationName)

	// construct MutatingWebhookConfiguration
	mwcWebhooks := []admissionregistrationv1.MutatingWebhook{}

	for _, w := range wr.Filter(MutatingWebhooksFilter()) {
		if !w.isInitialized {
			return nil, nil, fmt.Errorf("webhook '%s' is not initialized, please call Initialize on the webhook or InitializeAll on the registry before generating the MutatingWebhookConfiguration", w.Name)
		}
		whCfg, err := o.buildCommonWebhookConfiguration(w)
		if err != nil {
			return nil, nil, err
		}
		mwcWebhooks = append(mwcWebhooks, whCfg.toMutatingWebhook())
	}

	mwc.Webhooks = mwcWebhooks

	return mwc, func(obj client.Object) func() error {
		return func() error {
			typedObj, ok := obj.(*admissionregistrationv1.MutatingWebhookConfiguration)
			if !ok {
				return fmt.Errorf("update function for MutatingWebhookConfigurations called on invalid object")
			}
			typedObj.Webhooks = mwcWebhooks
			return nil
		}
	}, nil
}

func (o *ConfigOptions) buildCommonWebhookConfiguration(w *Webhook) (*commonWebhookConfig, error) {
	rule := admissionregistrationv1.RuleWithOperations{
		Operations: w.Operations,
		Rule:       admissionregistrationv1.Rule{},
	}
	rule.Rule.APIGroups = []string{w.APIGroup}
	rule.Rule.APIVersions = w.APIVersions
	rule.Rule.Resources = []string{w.ResourceName}
	clientConfig := admissionregistrationv1.WebhookClientConfig{
		CABundle: o.CABundle,
	}
	if o.WebhookURL != "" {
		parsedURL, err := url.Parse(o.WebhookURL)
		if err != nil {
			return nil, fmt.Errorf("unable to parse webhook url: %w", err)
		}
		parsedURL.Path = path.Join(parsedURL.Path, w.Path())
		webhookURL := parsedURL.String()
		clientConfig.URL = &webhookURL
	} else {
		webhookPath := w.Path()
		clientConfig.Service = &admissionregistrationv1.ServiceReference{
			Namespace: o.Service.Namespace,
			Name:      o.Service.Name,
			Path:      &webhookPath,
			Port:      &o.Service.Port,
		}
	}

	common := &commonWebhookConfig{
		Name:                    w.Name + o.WebhookNameSuffix,
		SideEffects:             utils.Ptr(admissionregistrationv1.SideEffectClassNone),
		FailurePolicy:           utils.Ptr(admissionregistrationv1.Fail),
		ObjectSelector:          w.LabelSelector,
		AdmissionReviewVersions: []string{"v1"},
		Rules:                   []admissionregistrationv1.RuleWithOperations{rule},
		ClientConfig:            clientConfig,
		TimeoutSeconds:          utils.Ptr(int32(w.Timeout)),
	}

	return common, nil
}
