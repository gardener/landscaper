// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/http"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lscore "github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	webhooklib "github.com/gardener/landscaper/controller-utils/pkg/webhook"
)

// INSTALLATION

var InstallationWebhookLogic webhooklib.WebhookLogic = func(ctx context.Context, req admission.Request, dec runtime.Decoder) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "InstallationWebhookLogic"})
	inst := &lscore.Installation{}
	if _, _, err := dec.Decode(req.Object.Raw, nil, inst); err != nil {
		logger.Debug("Decoding failed:" + err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateInstallation(inst); len(errs) > 0 {
		aggErr := errs.ToAggregate().Error()
		logger.Debug("Validation failed:" + aggErr)
		return admission.Denied(aggErr)
	}

	return admission.Allowed("Installation is valid")
}

// DEPLOYITEM

var DeployItemWebhookLogic webhooklib.WebhookLogic = func(ctx context.Context, req admission.Request, dec runtime.Decoder) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "DeployItemWebhookLogic"})

	di := &lscore.DeployItem{}
	if _, _, err := dec.Decode(req.Object.Raw, nil, di); err != nil {
		logger.Debug("Decoding failed:" + err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateDeployItem(di); len(errs) > 0 {
		aggErr := errs.ToAggregate().Error()
		logger.Debug("Validation failed:" + aggErr)
		return admission.Denied(aggErr)
	}

	// check if the type was updated for update events
	if req.Operation == admissionv1.Update {
		oldDi := &lscore.DeployItem{}
		if _, _, err := dec.Decode(req.OldObject.Raw, nil, oldDi); err != nil {
			logger.Debug("Decoding old failed:" + err.Error())
			return admission.Errored(http.StatusBadRequest, err)
		}
		if oldDi.Spec.Type != di.Spec.Type {
			logger.Debug(fmt.Sprintf("deployitem type is immutable, got %q but expected %q", di.Spec.Type, oldDi.Spec.Type))
			return admission.Errored(http.StatusForbidden, field.Forbidden(field.NewPath(".spec.type"), "type is immutable"))
		}
	}

	return admission.Allowed("DeployItem is valid")
}

// EXECUTION

var ExecutionWebhookLogic webhooklib.WebhookLogic = func(ctx context.Context, req admission.Request, dec runtime.Decoder) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "ExecutionWebhookLogic"})

	exec := &lscore.Execution{}
	if _, _, err := dec.Decode(req.Object.Raw, nil, exec); err != nil {
		logger.Debug("Decoding failed:" + err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateExecution(exec); len(errs) > 0 {
		aggErr := errs.ToAggregate().Error()
		logger.Debug("Validation failed:" + aggErr)
		return admission.Denied(aggErr)
	}

	return admission.Allowed("Execution is valid")
}

// TARGET

var TargetWebhookLogic webhooklib.WebhookLogic = func(ctx context.Context, req admission.Request, dec runtime.Decoder) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "ExecutionWebhookLogic"})

	t := &lscore.Target{}
	if _, _, err := dec.Decode(req.Object.Raw, nil, t); err != nil {
		logger.Debug("Decoding failed:" + err.Error())
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateTarget(t); len(errs) > 0 {
		aggErr := errs.ToAggregate().Error()
		logger.Debug("Validation failed:" + aggErr)
		return admission.Denied(aggErr)
	}

	return admission.Allowed("Target is valid")
}
