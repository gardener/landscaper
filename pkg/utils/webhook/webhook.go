// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	lc "github.com/gardener/landscaper/controller-utils/pkg/logging/constants"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lscore "github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// ValidatorFromResourceType is a helper method that gets a resource type and returns the fitting validator
func ValidatorFromResourceType(log logging.Logger, kubeClient client.Client, scheme *runtime.Scheme, resource string) (GenericValidator, error) {
	abstrVal := newAbstractedValidator(log, kubeClient, scheme)
	var val GenericValidator
	switch resource {
	case "installations":
		val = &InstallationValidator{abstrVal}
	case "deployitems":
		val = &DeployItemValidator{abstrVal}
	case "executions":
		val = &ExecutionValidator{abstrVal}
	case "targets":
		val = &TargetValidator{abstrVal}
	default:
		return nil, fmt.Errorf("unable to find validator for resource type %q", resource)
	}
	return val, nil
}

type abstractValidator struct {
	Client  client.Client
	decoder runtime.Decoder
	log     logging.Logger
}

// newAbstractedValidator creates a new abstracted validator
func newAbstractedValidator(log logging.Logger, kubeClient client.Client, scheme *runtime.Scheme) abstractValidator {
	return abstractValidator{
		Client:  kubeClient,
		decoder: serializer.NewCodecFactory(scheme).UniversalDecoder(),
		log:     log,
	}
}

// GenericValidator is an abstraction interface that implements admission.Handler and contains additional setter functions for the fields
type GenericValidator interface {
	Handle(context.Context, admission.Request) admission.Response
}

// INSTALLATION

// InstallationValidator represents a validator for an Installation
type InstallationValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (iv *InstallationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := iv.log.WithValues(lc.KeyResourceGroup, req.Kind.Group, lc.KeyResourceKind, req.Kind.Kind, lc.KeyResourceVersion, req.Kind.Version, lc.KeyResource, fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = logging.NewContext(ctx, logger)

	timeBefore := time.Now()
	result := iv.handlePrivate(ctx, req)

	logIfDurationExceeded(ctx, timeBefore)

	return result
}

func logIfDurationExceeded(ctx context.Context, timeBefore time.Time) {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "logIfDurationExceeded"})

	timeAfter := time.Now()
	diff := timeAfter.Sub(timeBefore)
	// check if request requires more than 9 seconds because 10 seconds timeout is configured at the landscaper webhook
	// calling this validation logic.
	if diff > 9*time.Second {
		logger.Info(fmt.Sprintf("validation request required more than 9 seconds: %s", diff.String()))
	}
}

func (iv *InstallationValidator) handlePrivate(ctx context.Context, req admission.Request) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "InstallationValidator.handlePrivate"})

	logger.Debug("Received request")

	inst := &lscore.Installation{}
	if _, _, err := iv.decoder.Decode(req.Object.Raw, nil, inst); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateInstallation(inst); len(errs) > 0 {
		errMsg := errs.ToAggregate().Error()
		logger.Info("Denied request", lc.KeyError, errMsg)
		return admission.Denied(errMsg)
	}

	logger.Info("Allowed request")
	return admission.Allowed("Installation is valid")
}

// DEPLOYITEM

// DeployItemValidator represents a validator for a DeployItem
type DeployItemValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (div *DeployItemValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := div.log.WithValues(lc.KeyResourceGroup, req.Kind.Group, lc.KeyResourceKind, req.Kind.Kind, lc.KeyResourceVersion, req.Kind.Version, lc.KeyResource, fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = logging.NewContext(ctx, logger)

	timeBefore := time.Now()
	result := div.handlePrivate(ctx, req)

	logIfDurationExceeded(ctx, timeBefore)

	return result
}

func (div *DeployItemValidator) handlePrivate(ctx context.Context, req admission.Request) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "DeployItemValidator.handlePrivate"})

	logger.Debug("Received request")

	di := &lscore.DeployItem{}
	if _, _, err := div.decoder.Decode(req.Object.Raw, nil, di); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateDeployItem(di); len(errs) > 0 {
		errMsg := errs.ToAggregate().Error()
		logger.Info("Denied request", lc.KeyError, errMsg)
		return admission.Denied(errMsg)
	}

	// check if the type was updated for update events
	if req.Operation == admissionv1.Update {
		oldDi := &lscore.DeployItem{}
		if _, _, err := div.decoder.Decode(req.OldObject.Raw, nil, oldDi); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if oldDi.Spec.Type != di.Spec.Type {
			logger.Debug(fmt.Sprintf("deployitem type is immutable, got %q but expected %q", di.Spec.Type, oldDi.Spec.Type))
			return admission.Errored(http.StatusForbidden, field.Forbidden(field.NewPath(".spec.type"), "type is immutable"))
		}
	}

	logger.Info("Allowed request")
	return admission.Allowed("DeployItem is valid")
}

// EXECUTION

// ExecutionValidator represents a validator for an Execution
type ExecutionValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (ev *ExecutionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := ev.log.WithValues(lc.KeyResourceGroup, req.Kind.Group, lc.KeyResourceKind, req.Kind.Kind, lc.KeyResourceVersion, req.Kind.Version, lc.KeyResource, fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = logging.NewContext(ctx, logger)

	timeBefore := time.Now()
	result := ev.handlePrivate(ctx, req)

	logIfDurationExceeded(ctx, timeBefore)

	return result
}

func (ev *ExecutionValidator) handlePrivate(ctx context.Context, req admission.Request) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "ExecutionValidator.handlePrivate"})

	logger.Debug("Received request")

	exec := &lscore.Execution{}
	if _, _, err := ev.decoder.Decode(req.Object.Raw, nil, exec); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateExecution(exec); len(errs) > 0 {
		errMsg := errs.ToAggregate().Error()
		logger.Info("Denied request", lc.KeyError, errMsg)
		return admission.Denied(errMsg)
	}

	logger.Info("Allowed request")
	return admission.Allowed("Execution is valid")
}

// TARGET

// TargetValidator represents a validator for a Target
type TargetValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (tv *TargetValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := tv.log.WithValues(lc.KeyResourceGroup, req.Kind.Group, lc.KeyResourceKind, req.Kind.Kind, lc.KeyResourceVersion, req.Kind.Version, lc.KeyResource, fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = logging.NewContext(ctx, logger)

	timeBefore := time.Now()
	result := tv.handlePrivate(ctx, req)

	logIfDurationExceeded(ctx, timeBefore)

	return result
}

func (tv *TargetValidator) handlePrivate(ctx context.Context, req admission.Request) admission.Response {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "TargetValidator.handlePrivate"})

	logger.Debug("Received request")

	t := &lscore.Target{}
	if _, _, err := tv.decoder.Decode(req.Object.Raw, nil, t); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateTarget(t); len(errs) > 0 {
		errMsg := errs.ToAggregate().Error()
		logger.Info("Denied request", lc.KeyError, errMsg)
		return admission.Denied(errMsg)
	}

	logger.Info("Allowed request")
	return admission.Allowed("Target is valid")
}
