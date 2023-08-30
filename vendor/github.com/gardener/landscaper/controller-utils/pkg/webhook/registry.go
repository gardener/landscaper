// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

var registry WebhookRegistry

// WebhookRegistry is a helper struct for listing webhooks which can be instantiated automatically from flags.
type WebhookRegistry map[string]*Webhook

// WebhookRegistryFilter is a filter function which can be used to filter the registry for webhooks with specific properties.
type WebhookRegistryFilter func(*Webhook) bool

// NewWebhookRegistry returns a new webhook registry.
// Alternatively, the Registry() function can be used to get a singleton registry.
func NewWebhookRegistry() WebhookRegistry {
	return WebhookRegistry(map[string]*Webhook{})
}

// Registry returns a singleton registry.
// Use NewWebhookRegistry instead if you want to run multiple webhook servers with different sets of webhooks or can't use a singleton for some other reason.
func Registry() WebhookRegistry {
	if registry == nil {
		registry = NewWebhookRegistry()
	}
	return registry
}

// Register is an alias for registry[webhook.name] = webhook.
// Returns the registry for chaining.
func (wr WebhookRegistry) Register(webhook *Webhook) WebhookRegistry {
	wr[webhook.Name] = webhook
	return wr
}

// GetEnabledWebhooks takes a set of webhook names and returns a registry containing only the webhooks whose names were NOT in the set.
// If the set contains an entry 'all', an empty registry is returned.
func (wr WebhookRegistry) GetEnabledWebhooks(disabled sets.Set[string]) WebhookRegistry {
	res := NewWebhookRegistry()
	if disabled.Has("all") {
		return res
	}
	for name, w := range wr {
		if !disabled.Has(name) {
			res.Register(w)
		}
	}
	return res
}

// Filter returns a new WebhookRegistry containing only the webhooks for which a filter returned true.
// Note that the filters are ORed. To AND filters, call the function multiples times: reg.Filter(f1).Filter(f2)
func (wr WebhookRegistry) Filter(filters ...WebhookRegistryFilter) WebhookRegistry {
	res := NewWebhookRegistry()
	for _, w := range wr {
		for _, f := range filters {
			if f(w) {
				res.Register(w)
				break
			}
		}
	}
	return res
}

// InitializeAll calls Initialize on all webhooks in the registry.
// No-op if the registry is empty.
func (wr WebhookRegistry) InitializeAll(ctx context.Context, scheme *runtime.Scheme) error {
	if len(wr) == 0 {
		return nil
	}
	log := logging.FromContextOrDiscard(ctx)
	for name, w := range wr {
		if err := w.Initialize(log, scheme); err != nil {
			return fmt.Errorf("error initializing webhook registered as '%s': %w", name, err)
		}
	}
	return nil
}

// AddToServer initializes all webhooks of the registry and registers them at the given webhook server.
// No-op if the webhook registry is empty or nil.
func (wr WebhookRegistry) AddToServer(ctx context.Context, webhookServer ctrlwebhook.Server, scheme *runtime.Scheme) error {
	log := logging.FromContextOrDiscard(ctx)

	if len(wr) == 0 {
		log.Info("AddToServer called on empty webhook registry, nothing to do")
		return nil
	}

	// registering webhooks
	for _, w := range wr {
		rsLogger := log.WithName(w.Name)

		if err := w.Initialize(rsLogger, scheme); err != nil {
			return fmt.Errorf("unable to initialize webhook '%s': %w", w.Name, err)
		}

		webhookPath := w.Path()
		rsLogger.Info("Registering webhook", "name", w.ResourceName, "path", webhookPath)
		admission := &ctrlwebhook.Admission{Handler: w}
		webhookServer.Register(webhookPath, admission)
	}

	return nil
}
