// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetCachelessClient is a helper function that returns a client that can be used before the manager is started.
// It calls all given installFuncs on the created scheme.
func GetCachelessClient(restConfig *rest.Config, installFuncs ...func(*runtime.Scheme)) (client.Client, error) {
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return nil, err
	}

	for _, f := range installFuncs {
		f(s)
	}

	return client.New(restConfig, client.Options{Scheme: s})
}

// ValidatingWebhooksFilter returns a predefined filter to be used with the WebhookRegistry's Filter method.
// The returned filter returns true for webhooks which have the type ValidatingWebhook.
func ValidatingWebhooksFilter() WebhookRegistryFilter {
	return func(w *Webhook) bool {
		return w.Type == ValidatingWebhook
	}
}

// MutatingWebhooksFilter returns a predefined filter to be used with the WebhookRegistry's Filter method.
// The returned filter returns true for webhooks which have the type MutatingWebhook.
func MutatingWebhooksFilter() WebhookRegistryFilter {
	return func(w *Webhook) bool {
		return w.Type == MutatingWebhook
	}
}

// LabelFilter returns a filter that can be used with the WebhookRegistry's Filter method.
// The returned filter returns true if the webhook has a corresponding label.
// If the second argument is non-nil, true is only returned if the label's value matches it,
// otherwise only existence of the label is checked.
func LabelFilter(key string, value *string) WebhookRegistryFilter {
	return func(w *Webhook) bool {
		if w.Labels == nil {
			return false
		}
		val, ok := w.Labels[key]
		if !ok {
			return false
		}
		if value != nil && val != *value {
			return false
		}
		return true
	}
}

// Not negates the result of the passed filter.
func Not(f WebhookRegistryFilter) WebhookRegistryFilter {
	return func(w *Webhook) bool {
		return !f(w)
	}
}
