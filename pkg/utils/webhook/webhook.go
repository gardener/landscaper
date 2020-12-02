// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"net/http"

	lscore "github.com/gardener/landscaper/apis/core"
	"github.com/gardener/landscaper/apis/core/validation"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type InstallationValidator struct {
	Client  client.Client
	decoder *admission.Decoder
	log     logr.Logger
}

func (iv *InstallationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	inst := &lscore.Installation{}
	if err := iv.decoder.Decode(req, inst); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateInstallation(inst); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("installation is valid")
}

func (iv *InstallationValidator) InjectDecoder(d *admission.Decoder) error {
	iv.decoder = d
	return nil
}

type DeployItemValidator struct {
	Client  client.Client
	decoder *admission.Decoder
	log     logr.Logger
}

func (div *DeployItemValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	di := &lscore.DeployItem{}
	if err := div.decoder.Decode(req, di); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateDeployItem(di); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("installation is valid")
}

func (div *DeployItemValidator) InjectDecoder(d *admission.Decoder) error {
	div.decoder = d
	return nil
}

type ExecutionValidator struct {
	Client  client.Client
	decoder *admission.Decoder
	log     logr.Logger
}

func (ev *ExecutionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	exec := &lscore.Execution{}
	if err := ev.decoder.Decode(req, exec); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if errs := validation.ValidateExecution(exec); len(errs) > 0 {
		return admission.Denied(errs.ToAggregate().Error())
	}

	return admission.Allowed("installation is valid")
}

func (ev *ExecutionValidator) InjectDecoder(d *admission.Decoder) error {
	ev.decoder = d
	return nil
}
