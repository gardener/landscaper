// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	"github.com/gardener/landscaper/controller-utils/pkg/webhook/certificates"
)

// WebhookNaming is a helper struct which stores naming rules for Validating-/MutatingWebhookConfigurations and their webhooks.
type WebhookNaming struct {
	// Name is used as the name of the Validating-/MutatingWebhookConfiguration.
	Name string
	// WebhookSuffix is appended to the webhook names from the registered webhooks.
	WebhookSuffix string
}

// ApplyWebhooksOptions is a helper struct to organize the arguments to the ApplyWebhooks function.
type ApplyWebhooksOptions struct {
	// NameValidating contains the naming rules for the ValidatingWebhookConfiguration, if any.
	// If nil, no ValidatingWebhookConfiguration will be created or deleted.
	NameValidating *WebhookNaming
	// NameMutating contains the naming rules for the MutatingWebhookConfiguration, if any.
	// If nil, no MutatingWebhookConfiguration will be created or deleted.
	NameMutating *WebhookNaming
	// Server is the webhook server to which the webhooks will be registered.
	Server ctrlwebhook.Server
	// Client is the k8s client, used to create the Validating-/MutatingWebhookConfiguration.
	Client client.Client
	// Registry is the webhook registry.
	// Will be filtered with the disabled webhooks provided in the WebhookFlags object.
	Registry WebhookRegistry
	// Flags contains the configuration received via CLI flags.
	// Note that the ApplyWebhooks function can only handle single clusters.
	// For multi-cluster configurations, call the ApplyWebhooks function multiple times and pass in a single-cluster configuration,
	// which can be generated by calling the WebhookFlag's ForSingleCluster method.
	Flags *WebhookFlags
	// CertName is the name to be used for the certificates.
	// Will also be used as name for the secret, with a "-certs" suffix, so it has to adhere to the k8s naming rules for secrets.
	// This field is only evaluated if the certificates are generated and not passed in.
	CertName string
	// CertDir is the path to the directory which stores the certificates.
	// This has to be the same that was given to the webhook server.
	// This field is only evaluated if the certificates are generated and not passed in.
	CertDir string
	// CACert is the CA certificate.
	// Can be passed in if the certificates have been generated beforehand, otherwise it will be generated.
	// Setting only one of CACert and ServerCert to nil is not supported at the moment, either both or none have to be specified.
	CACert *certificates.Certificate
	// ServerCert is the server certificate.
	// Can be passed in if the certificates have been generated beforehand, otherwise it will be generated.
	// Setting only one of CACert and ServerCert to nil is not supported at the moment, either both or none have to be specified.
	ServerCert *certificates.Certificate
}

// ApplyWebhooks is an auxiliary function which groups the commands that are usually required to make webhooks work in the cluster.
// This includes:
// - generating certificates, if the certificates in the options are nil
// - creating a MutatingWebhookConfiguration, if the registry contains any non-disabled mutating webhooks, deleting it otherwise
// - creating a ValidatingWebhookConfiguration, if the registry contains any non-disabled validating webhooks, deleting it otherwise
// - registering all non-disabled webhooks at the webhook server
//
// This function does not:
// - instantiate a webhook server, this has to happen before
// - start the webhook server, this has to be done afterwards
func ApplyWebhooks(ctx context.Context, opts *ApplyWebhooksOptions) error {
	baseLog := logging.FromContextOrDiscard(ctx)
	log := baseLog.WithName("webhookInit")

	if opts == nil {
		return fmt.Errorf("invalid ApplyWebhooks options: must not be nil")
	}
	if opts.Flags == nil {
		return fmt.Errorf("invalid ApplyWebhooks options: webhook flags must not be nil")
	}
	if opts.Flags.IsMultiCluster() {
		return fmt.Errorf("ApplyWebhooks can only handle single-cluster webhook flags")
	}

	// Certificate generation
	if opts.CACert == nil {
		log.Info("No certificates provided, checking for existing ones in the cluster or generating new ones")
		if opts.CertName == "" {
			return fmt.Errorf("invalid ApplyWebhooks options: cert name must be specified if the certificates are not provided")
		}
		var err error
		var dnsNames []string
		if opts.Flags.WebhookService != nil {
			dnsNames = GeDNSNamesFromNamespacedName(opts.Flags.WebhookService.Namespace, opts.Flags.WebhookService.Name)
		} else {
			if dnsNames, err = GetDNSNamesFromURL(opts.Flags.WebhookURL); err != nil {
				return fmt.Errorf("unable to create webhook certificate configuration: %w", err)
			}
		}
		secretName := fmt.Sprintf("%s-certs", opts.CertName)
		opts.CACert, opts.ServerCert, err = GenerateCertificates(ctx, opts.Client, opts.CertDir, opts.Flags.CertNamespace, opts.CertName, secretName, dnsNames)
		if err != nil {
			return fmt.Errorf("error during certificate generation: %w", err)
		}
	} else {
		log.Info("Certificates provided")
	}

	enabledWebhooks := opts.Registry.GetEnabledWebhooks(opts.Flags.DisabledWebhooks)

	// MutatingWebhookConfigurations
	if opts.NameMutating != nil {
		mutReg := enabledWebhooks.Filter(MutatingWebhooksFilter())

		mutRegCount := len(mutReg)
		if mutRegCount > 0 {
			log.Info("Mutating webhooks enabled, creating/updating MutatingWebhookConfiguration", "webhookCount", mutRegCount, lc.KeyResourceKind, "MutatingWebhookConfiguration", lc.KeyResource, opts.NameMutating.Name)

			if err := mutReg.InitializeAll(ctx, opts.Client.Scheme()); err != nil {
				return err
			}

			cfg := &ConfigOptions{
				WebhookConfigurationName: opts.NameMutating.Name,
				WebhookNameSuffix:        opts.NameMutating.WebhookSuffix,
				Service:                  opts.Flags.WebhookService,
				WebhookURL:               opts.Flags.WebhookURL,
				CABundle:                 opts.CACert.CertificatePEM,
			}
			if err := UpdateWebhookConfiguration(ctx, MutatingWebhook, opts.Client, mutReg, cfg); err != nil {
				return fmt.Errorf("error creating/updating MutatingWebhookConfiguration: %w", err)
			}
		} else {
			log.Info("Mutating webhooks disabled, deleting MutatingWebhookConfiguration", lc.KeyResourceKind, "MutatingWebhookConfiguration", lc.KeyResource, opts.NameMutating.Name)
			if err := DeleteWebhookConfiguration(ctx, MutatingWebhook, opts.Client, opts.NameMutating.Name); err != nil {
				return fmt.Errorf("error deleting MutatingWebhookConfiguration: %w", err)
			}
		}
	}

	// ValidatingWebhookConfigurations
	if opts.NameValidating != nil {
		valReg := enabledWebhooks.Filter(ValidatingWebhooksFilter())

		valRegCount := len(valReg)
		if valRegCount > 0 {
			log.Info("Validating webhooks enabled, creating/updating ValidatingWebhookConfiguration", "webhookCount", valRegCount, lc.KeyResourceKind, "ValidatingWebhookConfiguration", lc.KeyResource, opts.NameValidating.Name)

			if err := valReg.InitializeAll(ctx, opts.Client.Scheme()); err != nil {
				return err
			}

			cfg := &ConfigOptions{
				WebhookConfigurationName: opts.NameValidating.Name,
				WebhookNameSuffix:        opts.NameValidating.WebhookSuffix,
				Service:                  opts.Flags.WebhookService,
				WebhookURL:               opts.Flags.WebhookURL,
				CABundle:                 opts.CACert.CertificatePEM,
			}
			if err := UpdateWebhookConfiguration(ctx, ValidatingWebhook, opts.Client, valReg, cfg); err != nil {
				return fmt.Errorf("error creating/updating ValidatingWebhookConfiguration: %w", err)
			}
		} else {
			log.Info("Validating webhooks disabled, deleting ValidatingWebhookConfiguration", lc.KeyResourceKind, "ValidatingWebhookConfiguration", lc.KeyResource, opts.NameValidating.Name)
			if err := DeleteWebhookConfiguration(ctx, ValidatingWebhook, opts.Client, opts.NameValidating.Name); err != nil {
				return fmt.Errorf("error deleting ValidatingWebhookConfiguration: %w", err)
			}
		}
	}

	// Log webhooks
	for _, w := range enabledWebhooks {
		log.Info("Enabling webhook", "type", string(w.Type), "name", w.Name, "resourceName", w.ResourceName)
	}

	// Register webhooks at webhook server
	ctx = logging.NewContext(ctx, baseLog.WithName("webhook"))
	if err := enabledWebhooks.AddToServer(ctx, opts.Server, opts.Client.Scheme()); err != nil {
		return fmt.Errorf("unable to register webhooks at webhook server: %w", err)
	}

	return nil
}
