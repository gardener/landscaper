// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package extensions

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
)

type stdIOExecutable struct {
	bin  string
	args []string
	env  []string
}

// NewStdIOExecutable returns a resource processor extension which runs an executable in the
// background when calling Process(). It communicates with this processor via stdin/stdout pipes.
func NewStdIOExecutable(bin string, args []string, env map[string]string) (process.ResourceStreamProcessor, error) {
	parsedEnv := []string{}
	for k, v := range env {
		parsedEnv = append(parsedEnv, fmt.Sprintf("%s=%s", k, v))
	}

	e := stdIOExecutable{
		bin:  bin,
		args: args,
		env:  parsedEnv,
	}

	return &e, nil
}

func (e *stdIOExecutable) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	cmd := exec.CommandContext(ctx, e.bin, e.args...)
	cmd.Env = e.env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("unable to get stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("unable to get stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start processor: %w", err)
	}

	if _, err := io.Copy(stdin, r); err != nil {
		return fmt.Errorf("unable to write input: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("unable to close input writer: %w", err)
	}

	if _, err := io.Copy(w, stdout); err != nil {
		return fmt.Errorf("unable to read output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("unable to wait for processor: %w", err)
	}

	return nil
}
