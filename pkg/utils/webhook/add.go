// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebhookedResourceDefinition contains information about the resources that should be watched by the webhook
type WebhookedResourceDefinition struct {
	// APIGroup of the resource
	APIGroup string
	// all APIVersions of the resource that should be handled
	APIVersions []string
	// name of the resource, lower-case plural form
	ResourceName string
}

// Options contains the configuration that is necessary to create a ValidatingWebhookConfiguration
type Options struct {
	// Name of the ValidatingWebhookConfiguration that will be created
	WebhookConfigurationName string
	// the webhooks will be named <resource><webhook suffix>
	WebhookNameSuffix string
	// base path for the webhooks, the resource name will be appended
	WebhookBasePath string
	// name of the service under which the webhook can be reached
	ServiceName string
	// namespace of the service
	ServiceNamespace string
	// port of the service
	ServicePort int32
	// external service URL
	WebhookURL string
	// LabelSelector that is used to filter all resources handled by this webhook
	ObjectSelector metav1.LabelSelector
	// the resources that should be handled by this webhook
	WebhookedResources []WebhookedResourceDefinition
	// certificates for the webhook
	CABundle []byte
}

// UpdateValidatingWebhookConfiguration will create or update a ValidatingWebhookConfiguration
func UpdateValidatingWebhookConfiguration(ctx context.Context, kubeClient client.Client, o Options, webhookLogger logging.Logger) error {
	// do not deploy or update the webhook if no service name or webhook url is given
	if (len(o.ServiceName) == 0 || len(o.ServiceNamespace) == 0) && len(o.WebhookURL) == 0 {
		return nil
	}

	vwc := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.WebhookConfigurationName,
		},
	}

	// construct ValidatingWebhookConfiguration
	noSideEffects := admissionregistrationv1.SideEffectClassNone
	failPolicy := admissionregistrationv1.Fail
	vwcWebhooks := []admissionregistrationv1.ValidatingWebhook{}

	for _, elem := range o.WebhookedResources {
		rule := admissionregistrationv1.RuleWithOperations{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
			Rule:       admissionregistrationv1.Rule{},
		}
		rule.Rule.APIGroups = []string{elem.APIGroup}
		rule.Rule.APIVersions = elem.APIVersions
		rule.Rule.Resources = []string{elem.ResourceName}
		clientConfig := admissionregistrationv1.WebhookClientConfig{
			CABundle: o.CABundle,
		}
		if len(o.WebhookURL) != 0 {
			parsedURL, err := url.Parse(o.WebhookURL)
			if err != nil {
				return fmt.Errorf("unable to parse webhook url: %w", err)
			}
			parsedURL.Path = path.Join(parsedURL.Path, o.WebhookBasePath, elem.ResourceName)
			webhookURL := parsedURL.String()
			clientConfig.URL = &webhookURL
		} else {
			webhookPath := path.Join(o.WebhookBasePath, elem.ResourceName)
			clientConfig.Service = &admissionregistrationv1.ServiceReference{
				Namespace: o.ServiceNamespace,
				Name:      o.ServiceName,
				Path:      &webhookPath,
				Port:      &o.ServicePort,
			}
		}
		vwcWebhook := admissionregistrationv1.ValidatingWebhook{
			Name:                    elem.ResourceName + o.WebhookNameSuffix,
			SideEffects:             &noSideEffects,
			FailurePolicy:           &failPolicy,
			ObjectSelector:          &o.ObjectSelector,
			AdmissionReviewVersions: []string{"v1"},
			Rules:                   []admissionregistrationv1.RuleWithOperations{rule},
			ClientConfig:            clientConfig,
		}
		vwcWebhooks = append(vwcWebhooks, vwcWebhook)
	}

	webhookLogger.Info("Creating/updating ValidatingWebhookConfiguration", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
	_, err := ctrl.CreateOrUpdate(ctx, kubeClient, &vwc, func() error {
		vwc.Webhooks = vwcWebhooks
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to create/update ValidatingWebhookConfiguration: %w", err)
	}
	webhookLogger.Info("ValidatingWebhookConfiguration created/updated", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")

	return nil
}

// DeleteValidatingWebhookConfiguration deletes a ValidatingWebhookConfiguration
func DeleteValidatingWebhookConfiguration(ctx context.Context, kubeClient client.Client, name string, webhookLogger logging.Logger) error {
	vwc := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	webhookLogger.Info("Removing ValidatingWebhookConfiguration, if it exists", "name", name, "kind", "ValidatingWebhookConfiguration")
	if err := kubeClient.Delete(ctx, &vwc); err != nil {
		if apierrors.IsNotFound(err) {
			webhookLogger.Debug("ValidatingWebhookConfiguration not found", "name", name, "kind", "ValidatingWebhookConfiguration")
		} else {
			return fmt.Errorf("unable to delete ValidatingWebhookConfiguration %q: %w", name, err)
		}
	} else {
		webhookLogger.Info("ValidatingWebhookConfiguration deleted", "name", name, "kind", "ValidatingWebhookConfiguration")
	}
	return nil
}

// RegisterWebhooks generates certificates and registers the webhooks to the manager
// no-op if WebhookedResources in the given options is either nil or empty
func RegisterWebhooks(log logging.Logger, webhookServer *ctrlwebhook.Server, client client.Client, scheme *runtime.Scheme, o Options) error {
	if o.WebhookedResources == nil || len(o.WebhookedResources) == 0 {
		return nil
	}

	// registering webhooks
	for _, elem := range o.WebhookedResources {
		rsLogger := log.WithName(elem.ResourceName)
		val, err := ValidatorFromResourceType(rsLogger, client, scheme, elem.ResourceName)
		if err != nil {
			return fmt.Errorf("unable to register webhooks: %w", err)
		}

		webhookPath := o.WebhookBasePath + elem.ResourceName
		rsLogger.Info("Registering webhook", "resource", elem.ResourceName, "path", webhookPath)
		admission := &ctrlwebhook.Admission{Handler: val}
		_ = admission.InjectLogger(rsLogger.Logr())
		if err := admission.InjectScheme(scheme); err != nil {
			return err
		}
		webhookServer.Register(webhookPath, admission)
	}

	return nil
}
