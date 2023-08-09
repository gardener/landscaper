// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	install "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/pkg/version"

	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
	webhooklib "github.com/gardener/landscaper/controller-utils/pkg/webhook"
)

func NewLandscaperWebhooksCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "landscaper-webhooks",
		Short: "Landscaper webhooks serves the landscaper validation, mutating and defaulting webhooks.",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.log.Error(err, "unable to run landscaper webhooks server")
				os.Exit(1)
			}
		},
	}
	options.AddFlags(cmd.Flags())
	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.log.Info("Starting Landscaper Webhooks Server", lc.KeyVersion, version.Get().String())
	ctx = logging.NewContext(ctx, o.log)

	opts := ctrlwebhook.Options{}
	opts.Port = o.webhookConfig.Port
	opts.WebhookMux = http.NewServeMux()
	opts.WebhookMux.Handle("/healthz", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		if _, err := writer.Write([]byte("Ok")); err != nil {
			o.log.Error(err, "unable to send health response")
		}
	}))
	opts.CertDir = filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	webhookServer := ctrlwebhook.NewServer(opts)

	ctrl.SetLogger(o.log.Logr())

	restConfig := ctrl.GetConfigOrDie()
	scheme := runtime.NewScheme()
	utilruntime.Must(kubernetesscheme.AddToScheme(scheme))
	install.Install(scheme)

	kubeClient, err := webhooklib.GetCachelessClient(restConfig, install.Install)
	if err != nil {
		return fmt.Errorf("unable to get client: %w", err)
	}

	if err := webhooklib.ApplyWebhooks(ctx, &webhooklib.ApplyWebhooksOptions{
		NameValidating: &webhooklib.WebhookNaming{
			Name:          "landscaper-validation-webhook",
			WebhookSuffix: ".validation.landscaper.gardener.cloud",
		},
		Server:   webhookServer,
		Client:   kubeClient,
		Registry: defaultWebhooks,
		Flags:    o.webhookConfig,
		CertName: "landscaper-webhook",
		CertDir:  opts.CertDir,
	}); err != nil {
		return fmt.Errorf("error applying the webhooks: %w", err)
	}

	if err := webhookServer.Start(ctx); err != nil {
		panic(fmt.Errorf("error while starting webhook server: %w", err))
	}
	return nil
}
