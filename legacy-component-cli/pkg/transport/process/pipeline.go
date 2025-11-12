// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"io"
	"os"
	"time"

	"fmt"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/utils"
)

const processorTimeout = 30 * time.Second

type resourceProcessingPipelineImpl struct {
	processors []ResourceStreamProcessor
}

func (p *resourceProcessingPipelineImpl) Process(ctx context.Context, cd cdv2.ComponentDescriptor, res cdv2.Resource) (*cdv2.ComponentDescriptor, cdv2.Resource, error) {
	infile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to create temporary infile: %w", err)
	}

	if err := utils.WriteProcessorMessage(cd, res, nil, infile); err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to write: %w", err)
	}

	for _, proc := range p.processors {
		outfile, err := p.runProcessor(ctx, infile, proc)
		if err != nil {
			return nil, cdv2.Resource{}, err
		}

		infile = outfile
	}
	defer infile.Close()

	if _, err := infile.Seek(0, io.SeekStart); err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to seek to beginning of input file: %w", err)
	}

	processedCD, processedRes, blobreader, err := utils.ReadProcessorMessage(infile)
	if err != nil {
		return nil, cdv2.Resource{}, fmt.Errorf("unable to read output data: %w", err)
	}
	if blobreader != nil {
		defer blobreader.Close()
	}

	return processedCD, processedRes, nil
}

func (p *resourceProcessingPipelineImpl) runProcessor(ctx context.Context, infile *os.File, proc ResourceStreamProcessor) (*os.File, error) {
	defer infile.Close()

	if _, err := infile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("unable to seek to beginning of input file: %w", err)
	}

	outfile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary outfile: %w", err)
	}

	inreader := infile
	outwriter := outfile

	ctx, cancelfunc := context.WithTimeout(ctx, processorTimeout)
	defer cancelfunc()

	if err := proc.Process(ctx, inreader, outwriter); err != nil {
		return nil, fmt.Errorf("unable to process resource: %w", err)
	}

	return outfile, nil
}

// NewResourceProcessingPipeline returns a new ResourceProcessingPipeline
func NewResourceProcessingPipeline(processors ...ResourceStreamProcessor) ResourceProcessingPipeline {
	p := resourceProcessingPipelineImpl{
		processors: processors,
	}
	return &p
}
