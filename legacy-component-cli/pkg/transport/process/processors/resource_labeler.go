// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package processors

import (
	"context"
	"fmt"
	"io"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

type resourceLabeler struct {
	labels cdv2.Labels
}

// NewResourceLabeler returns a processor that appends one or more labels to a resource
func NewResourceLabeler(labels ...cdv2.Label) process.ResourceStreamProcessor {
	obj := resourceLabeler{
		labels: labels,
	}
	return &obj
}

func (p *resourceLabeler) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cd, res, resBlobReader, err := utils.ReadProcessorMessage(r)
	if err != nil {
		return fmt.Errorf("unable to read processor message: %w", err)
	}
	if resBlobReader != nil {
		defer resBlobReader.Close()
	}

	res.Labels = append(res.Labels, p.labels...)

	if err := utils.WriteProcessorMessage(*cd, res, resBlobReader, w); err != nil {
		return fmt.Errorf("unable to write processor message: %w", err)
	}

	return nil
}
