// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"io"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

// ResourceProcessingPipeline describes a chain of multiple processors for processing a resource.
// Each processor receives its input from the preceding processor and writes the output for the
// subsequent processor. To work correctly, a pipeline must consist of 1 downloader, 0..n processors,
// and 1..n uploaders.
type ResourceProcessingPipeline interface {
	// Process executes all processors for a resource.
	// Returns the component descriptor and resource of the last processor.
	Process(context.Context, cdv2.ComponentDescriptor, cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error)
}

// ResourceStreamProcessor describes an individual processor for processing a resource.
// A processor can upload, modify, or download a resource.
type ResourceStreamProcessor interface {
	// Process executes the processor for a resource. Input and Output streams must be
	// compliant to a specific format ("processor message"). See also ./utils/processor_message.go
	// which describes the format and provides helper functions to read/write processor messages.
	Process(context.Context, io.Reader, io.Writer) error
}
