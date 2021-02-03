// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

///// WEBHOOK /////

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
	// LabelSelector that is used to filter all resources handled by this webhook
	ObjectSelector metav1.LabelSelector
	// the resources that should be handled by this webhook
	WebhookedResources []WebhookedResourceDefinition
	// certificates for the webhook
	CABundle []byte
}

// ApplyValidatingWebhookConfiguration will create, update, or delete a ValidatingWebhookConfiguration, depending on the options
// If WebhookedResources in the given options is neither nil nor empty, a ValidatingWebhookConfiguration will be created/updated
// otherwise it will be deleted, if it exists.
func ApplyValidatingWebhookConfiguration(ctx context.Context, mgr manager.Manager, o Options, webhookLogger logr.Logger) error {
	tmpClient, err := getCachelessClient(mgr)
	if err != nil {
		return fmt.Errorf("unable to get client: %w", err)
	}

	vwc := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.WebhookConfigurationName,
		},
	}

	if o.WebhookedResources != nil && len(o.WebhookedResources) > 0 {
		webhookLogger.Info("Validation webhook enabled")
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
			webhookPath := o.WebhookBasePath + elem.ResourceName
			vwcWebhook := admissionregistrationv1.ValidatingWebhook{
				Name:                    elem.ResourceName + o.WebhookNameSuffix,
				SideEffects:             &noSideEffects,
				FailurePolicy:           &failPolicy,
				ObjectSelector:          &o.ObjectSelector,
				AdmissionReviewVersions: []string{"v1"},
				Rules:                   []admissionregistrationv1.RuleWithOperations{rule},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: o.ServiceNamespace,
						Name:      o.ServiceName,
						Path:      &webhookPath,
						Port:      &o.ServicePort,
					},
					CABundle: o.CABundle,
				},
			}
			vwcWebhooks = append(vwcWebhooks, vwcWebhook)
		}

		webhookLogger.Info("Creating ValidatingWebhookConfiguration", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
		_, err = ctrl.CreateOrUpdate(ctx, tmpClient, &vwc, func() error {
			vwc.Webhooks = vwcWebhooks
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to create/update ValidatingWebhookConfiguration: %w", err)
		}
		webhookLogger.Info("ValidatingWebhookConfiguration created/updated", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
	} else {
		webhookLogger.Info("Validation webhook disabled")
		webhookLogger.Info("Removing ValidatingWebhookConfiguration, if it exists", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
		if err := tmpClient.Delete(ctx, &vwc); err != nil {
			if apierrors.IsNotFound(err) {
				webhookLogger.Info("ValidatingWebhookConfiguration not found", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
			} else {
				webhookLogger.Error(err, "unable to delete validatingwebhookconfiguration", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
			}
		} else {
			webhookLogger.Info("ValidatingWebhookConfiguration deleted", "name", o.WebhookConfigurationName, "kind", "ValidatingWebhookConfiguration")
		}
	}

	return nil
}

// RegisterWebhooks generates certificates and registers the webhooks to the manager
// no-op if WebhookedResources in the given options is either nil or empty
func RegisterWebhooks(ctx context.Context, mgr manager.Manager, o Options, webhookLogger logr.Logger) error {
	if o.WebhookedResources == nil || len(o.WebhookedResources) == 0 {
		return nil
	}

	// registering webhooks
	for _, elem := range o.WebhookedResources {
		val, err := ValidatorFromResourceType(elem.ResourceName)
		if err != nil {
			return fmt.Errorf("unable to register webhooks: %w", err)
		}
		if err := val.InjectClient(mgr.GetClient()); err != nil {
			return fmt.Errorf("unable to register webhooks: %w", err)
		}
		rsLogger := webhookLogger.WithName(elem.ResourceName)
		webhookPath := o.WebhookBasePath + elem.ResourceName
		if err := val.InjectLogger(rsLogger); err != nil {
			return fmt.Errorf("unable to register webhooks: %w", err)
		}

		rsLogger.Info("Registering webhook", "resource", elem.ResourceName, "path", webhookPath)
		mgr.GetWebhookServer().Register(webhookPath, &ctrlwebhook.Admission{Handler: val})
	}

	return nil
}
