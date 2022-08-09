// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

// ValidatorFromResourceType is a helper method that gets a resource type and returns the fitting validator
func ValidatorFromResourceType(log logging.Logger, kubeClient client.Client, scheme *runtime.Scheme, resource string) (GenericValidator, error) {
	abstrVal := newAbstractedValidator(log, kubeClient, scheme)
	var val GenericValidator
	if resource == "installations" {
		val = &InstallationValidator{abstrVal}
	} else if resource == "deployitems" {
		val = &DeployItemValidator{abstrVal}
	} else if resource == "executions" {
		val = &ExecutionValidator{abstrVal}
	} else if resource == "componentoverwrites" {
		val = &ComponentOverwritesValidator{abstrVal}
	} else {
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
	iv.log.Debug("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	inst := &lscore.Installation{}
	if _, _, err := iv.decoder.Decode(req.Object.Raw, nil, inst); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateInstallation(inst); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	// check if the installation declares an export which is already exported by another installation in the same context
	v1alpha1Inst := &lsv1alpha1.Installation{}
	err := lsv1alpha1.Convert_core_Installation_To_v1alpha1_Installation(inst, v1alpha1Inst, nil)
	if err != nil {
		iv.log.Error(err, "error while converting core Installation to v1alpha1 Installation")
	} else {
		dupErr, err := checkForDuplicateExports(ctx, iv.Client, v1alpha1Inst)
		if err != nil {
			iv.log.Error(err, "error while checking for duplicate exports")
		} else if dupErr != nil {
			return admission.Denied(dupErr.Error())
		}
	}

	return admission.Allowed("Installation is valid")
}

// DEPLOYITEM

// DeployItemValidator represents a validator for a DeployItem
type DeployItemValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (div *DeployItemValidator) Handle(_ context.Context, req admission.Request) admission.Response {
	div.log.Debug("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	di := &lscore.DeployItem{}
	if _, _, err := div.decoder.Decode(req.Object.Raw, nil, di); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateDeployItem(di); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	// check if the type was updated for update events
	if req.Operation == admissionv1.Update {
		oldDi := &lscore.DeployItem{}
		if _, _, err := div.decoder.Decode(req.OldObject.Raw, nil, oldDi); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if oldDi.Spec.Type != di.Spec.Type {
			div.log.Debug(fmt.Sprintf("deployitem type is immutable, got %q but expected %q", di.Spec.Type, oldDi.Spec.Type))
			return admission.Errored(http.StatusForbidden, field.Forbidden(field.NewPath(".spec.type"), "type is immutable"))
		}
	}

	return admission.Allowed("DeployItem is valid")
}

// EXECUTION

// ExecutionValidator represents a validator for an Execution
type ExecutionValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (ev *ExecutionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ev.log.Debug("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	exec := &lscore.Execution{}
	if _, _, err := ev.decoder.Decode(req.Object.Raw, nil, exec); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateExecution(exec); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("Execution is valid")
}

// EXECUTION

// ComponentOverwritesValidator represents a validator for ComponentOverwrites
type ComponentOverwritesValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (ev *ComponentOverwritesValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ev.log.Debug("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	co := &lscore.ComponentOverwrites{}
	if _, _, err := ev.decoder.Decode(req.Object.Raw, nil, co); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateComponentOverwrites(co); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("ComponentOverwrite is valid")
}
