// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"

	"github.com/gardener/landscaper/apis/core"

	flag "github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	webhooklib "github.com/gardener/landscaper/controller-utils/pkg/webhook"
	webhook "github.com/gardener/landscaper/pkg/utils/webhook"
)

var landscaperSkipValidationSelector = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{
		{
			Operator: metav1.LabelSelectorOpNotIn,
			Key:      "validation.landscaper.gardener.cloud/skip-validation",
			Values:   []string{"true"},
		},
	},
}

var defaultWebhooks = webhooklib.NewWebhookRegistry().
	Register(&webhooklib.Webhook{
		Name:          "installations",
		Type:          webhooklib.ValidatingWebhook,
		APIGroup:      core.GroupName,
		APIVersions:   []string{"v1alpha1"},
		ResourceName:  "installations",
		Operations:    webhooklib.Operations(webhooklib.CREATE, webhooklib.UPDATE),
		LabelSelector: landscaperSkipValidationSelector,
		Process:       webhook.InstallationWebhookLogic,
	}).
	Register(&webhooklib.Webhook{
		Name:          "deployitems",
		Type:          webhooklib.ValidatingWebhook,
		APIGroup:      core.GroupName,
		APIVersions:   []string{"v1alpha1"},
		ResourceName:  "deployitems",
		Operations:    webhooklib.Operations(webhooklib.CREATE, webhooklib.UPDATE),
		LabelSelector: landscaperSkipValidationSelector,
		Process:       webhook.DeployItemWebhookLogic,
	}).
	Register(&webhooklib.Webhook{
		Name:          "executions",
		Type:          webhooklib.ValidatingWebhook,
		APIGroup:      core.GroupName,
		APIVersions:   []string{"v1alpha1"},
		ResourceName:  "executions",
		Operations:    webhooklib.Operations(webhooklib.CREATE, webhooklib.UPDATE),
		LabelSelector: landscaperSkipValidationSelector,
		Process:       webhook.ExecutionWebhookLogic,
	}).
	Register(&webhooklib.Webhook{
		Name:          "targets",
		Type:          webhooklib.ValidatingWebhook,
		APIGroup:      core.GroupName,
		APIVersions:   []string{"v1alpha1"},
		ResourceName:  "targets",
		Operations:    webhooklib.Operations(webhooklib.CREATE, webhooklib.UPDATE),
		LabelSelector: landscaperSkipValidationSelector,
		Process:       webhook.TargetWebhookLogic,
	})

type options struct {
	log           logging.Logger
	webhookConfig *webhooklib.WebhookFlags
}

func NewOptions() *options {
	return &options{
		webhookConfig: webhooklib.NewWebhookFlags(),
	}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	o.webhookConfig.AddFlags(fs)
	logging.InitFlags(fs)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.log = log

	err = o.webhookConfig.Complete(defaultWebhooks)
	if err != nil {
		return err
	}

	return nil
}
