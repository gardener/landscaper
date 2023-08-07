// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"path"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"
)

const (
	JSONPatchAddOperation     = "add"
	JSONPatchReplaceOperation = "replace"
	JSONPatchRemoveOperation  = "remove"

	DefaultWebhookTimeoutSeconds = 15

	// forwarded for easier consumption
	CREATE = admissionregistrationv1.Create
	UPDATE = admissionregistrationv1.Update
	DELETE = admissionregistrationv1.Delete

	ValidatingWebhookBasePath = "/webhook/validate"
	MutatingWebhookBasePath   = "/webhook/mutate"
)

// Operations returns the given arguments as slice.
// This is an auxiliary function which removes the need for importing admissionregistration/v1 in the calling file.
func Operations(ops ...admissionregistrationv1.OperationType) []admissionregistrationv1.OperationType {
	return ops
}

// AllOperations is an alias for Operations(CREATE, UPDATE, DELETE)
func AllOperations() []admissionregistrationv1.OperationType {
	return Operations(CREATE, UPDATE, DELETE)
}

type WebhookType string

const (
	ValidatingWebhook WebhookType = "ValidatingWebhook"
	MutatingWebhook   WebhookType = "MutatingWebhook"
)

// WebhookLogic represents the actual logic of a webhook.
// It is similar to admission.Handler.Handle, but in addition, a default decoder is passed in.
// The decoder is derived from the scheme which was passed in during webhook initialization.
type WebhookLogic func(context.Context, admission.Request, runtime.Decoder) admission.Response

var _ admission.Handler = &Webhook{}

type Webhook struct {
	// (usually) statically defined fields
	// Name is the name of the webhook. Used for logging, webhook path, and inside Validating-/MutatingWebhookConfigurations.
	Name string
	// Type is the type, whether the webhook is a validating or a mutating webhook.
	Type WebhookType
	// APIGroup is the api group of the resource watched by this webhook.
	APIGroup string
	// APIVersions are the api versions watched by this webhook.
	APIVersions []string
	// ResourceName is the name of the resource watched by this webhook.
	ResourceName string
	// Operations lists the operations on which this webhook should react.
	Operations []admissionregistrationv1.OperationType
	// Timeout is the timeout for the webhook in seconds.
	Timeout int
	// Process is the function which holds the actual webhook logic.
	// Note that it is wrapped in some additional logic which logs the result.
	Process WebhookLogic
	// Labels is a custom set of labels.
	// This field is not evaluated by the webhook library.
	// It can be used to group webhooks in combination with the WebhookRegistry's 'Filter' method.
	Labels map[string]string
	// LabelSelector allows to filter resources by labels.
	LabelSelector *metav1.LabelSelector

	// dynamic fields
	// Decoder is used to decode the raw object in the webhook's request into a structured object.
	Decoder runtime.Decoder
	// Log is the logger.
	Log logging.Logger

	isInitialized bool
}

// Initialize validates the webhook fields, defaults some of them and sets the dynamic fields.
// Is a no-op if called more than once.
func (w *Webhook) Initialize(log logging.Logger, scheme *runtime.Scheme) error {
	if w.isInitialized {
		return nil
	}

	if w.Name == "" {
		return fmt.Errorf("webhook name must not be empty")
	}
	if w.Type == "" {
		return fmt.Errorf("webhook type must not be empty")
	}
	if len(w.APIVersions) == 0 {
		return fmt.Errorf("no api versions specified")
	}
	if w.ResourceName == "" {
		return fmt.Errorf("webhook resource name must not be empty")
	}
	if len(w.Operations) == 0 {
		w.Operations = AllOperations()
	}
	if w.Timeout < 1 || w.Timeout > 30 {
		w.Timeout = DefaultWebhookTimeoutSeconds
	}
	if w.Process == nil {
		return fmt.Errorf("webhook's Process method must not be nil")
	}

	w.Decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	w.Log = log

	w.isInitialized = true

	return nil
}

func (w *Webhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log, ctx := w.Log.WithValuesAndContext(ctx, lc.KeyResourceGroup, req.Kind.Group, lc.KeyResourceKind, req.Kind.Kind, lc.KeyResourceVersion, req.Kind.Version, "event", string(req.Operation), lc.KeyResource, fmt.Sprintf("%s/%s", req.Namespace, req.Name))

	log.Debug("Request received")

	timeBefore := time.Now()
	res := w.Process(ctx, req, w.Decoder)

	logIfDurationExceeded(log, w.Timeout, timeBefore)

	msg := ""
	if res.Result != nil && string(res.Result.Message) != "" {
		msg = string(res.Result.Message)
	}

	if res.Allowed {
		if w.Type == MutatingWebhook {
			log = log.WithValues("patched", len(res.Patches) > 0)
		}
		log.Info("Request allowed", lc.KeyReason, msg)
	} else {
		log.Info("Request denied", lc.KeyReason, msg)
	}
	return res
}

// Path returns the path under which this webhook can be reached.
// It's a combination of the base path and the webhook's name.
func (w *Webhook) Path() string {
	switch w.Type {
	case ValidatingWebhook:
		return path.Join(ValidatingWebhookBasePath, w.Name)
	case MutatingWebhook:
		return path.Join(MutatingWebhookBasePath, w.Name)
	}
	return ""
}

func logIfDurationExceeded(log logging.Logger, timeout int, timeBefore time.Time) {
	timeAfter := time.Now()
	diff := timeAfter.Sub(timeBefore)
	if diff >= time.Duration(timeout)*time.Second {
		log.Info("Webhook request took more time than the defined timeout", "timeout", time.Duration(timeout).String(), "duration", diff.String())
	}
}
