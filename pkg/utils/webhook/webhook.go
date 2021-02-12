// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	lscore "github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
)

// ValidatorFromResourceType is a helper method that gets a resource type and returns the fitting validator
func ValidatorFromResourceType(resource string) (GenericValidator, error) {
	var val GenericValidator
	if resource == "installations" {
		val = &InstallationValidator{}
	} else if resource == "deployitems" {
		val = &DeployItemValidator{}
	} else if resource == "executions" {
		val = &ExecutionValidator{}
	} else {
		return nil, fmt.Errorf("unable to find validator for resource type %q", resource)
	}
	return val, nil
}

type abstractValidator struct {
	Client  client.Client
	decoder runtime.Decoder
	log     logr.Logger
}

// GenericValidator is an abstraction interface that implements admission.Handler and contains additional setter functions for the fields
type GenericValidator interface {
	Handle(context.Context, admission.Request) admission.Response
	InjectScheme(*runtime.Scheme) error
	InjectClient(client.Client) error
	InjectLogger(logr.Logger) error
}

func (av *abstractValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	return admission.Denied("call to abstract method Handle, please implement")
}

func (av *abstractValidator) InjectScheme(scheme *runtime.Scheme) error {
	av.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (av *abstractValidator) InjectClient(c client.Client) error {
	av.Client = c
	return nil
}

func (av *abstractValidator) InjectLogger(l logr.Logger) error {
	av.log = l
	return nil
}

// INSTALLATION

// InstallationValidator represents a validator for an Installation
type InstallationValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (iv *InstallationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	iv.log.V(5).Info("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	inst := &lscore.Installation{}
	if _, _, err := iv.decoder.Decode(req.Object.Raw, nil, inst); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateInstallation(inst); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("Installation is valid")
}

// DEPLOYITEM

// DeployItemValidator represents a validator for a DeployItem
type DeployItemValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (div *DeployItemValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	div.log.V(5).Info("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	di := &lscore.DeployItem{}
	if _, _, err := div.decoder.Decode(req.Object.Raw, nil, di); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateDeployItem(di); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("DeployItem is valid")
}

// EXECUTION

// ExecutionValidator represents a validator for an Execution
type ExecutionValidator struct{ abstractValidator }

// Handle handles a request to the webhook
func (ev *ExecutionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	ev.log.V(5).Info("Received request", "group", req.Kind.Group, "kind", req.Kind.Kind, "version", req.Kind.Version)
	exec := &lscore.Execution{}
	if _, _, err := ev.decoder.Decode(req.Object.Raw, nil, exec); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateExecution(exec); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("Execution is valid")
}
