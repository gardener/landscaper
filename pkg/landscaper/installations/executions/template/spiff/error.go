// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package spiff

import (
	"strings"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

// TemplateError wraps a spiff templating error and adds more human-readable information.
type TemplateError struct {
	err            error
	inputFormatter *template.TemplateInputFormatter

	message string
}

// TemplateErrorBuilder creates a new TemplateError.
func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err: err,
	}
}

// WithInputFormatter adds a template input formatter to the error.
func (e *TemplateError) WithInputFormatter(inputFormatter *template.TemplateInputFormatter) *TemplateError {
	e.inputFormatter = inputFormatter
	return e
}

// Build builds the error message.
func (e *TemplateError) Build() *TemplateError {
	builder := strings.Builder{}
	builder.WriteString(e.err.Error())

	if e.inputFormatter != nil {
		builder.WriteString("\ntemplate input:\n")
		builder.WriteString(e.inputFormatter.Format("\t"))
	}

	e.message = builder.String()
	return e
}

// Error returns the error message.
func (e *TemplateError) Error() string {
	return e.message
}
