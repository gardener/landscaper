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
	input          map[string]interface{}
	inputFormatter *template.TemplateInputFormatter
	message        string
}

// TemplateErrorBuilder creates a new TemplateError.
func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err: err,
	}
}

// WithInput adds the template input with a formatter to the error.
func (e *TemplateError) WithInput(input map[string]interface{}, inputFormatter *template.TemplateInputFormatter) *TemplateError {
	e.input = input
	e.inputFormatter = inputFormatter
	return e
}

// Build builds the error message.
func (e *TemplateError) Build() *TemplateError {
	builder := strings.Builder{}
	builder.WriteString(e.err.Error())

	if e.input != nil && e.inputFormatter != nil {
		builder.WriteString("\ntemplate input:\n")
		builder.WriteString(e.inputFormatter.Format(e.input, "\t"))
	}

	e.message = builder.String()
	return e
}

// Error returns the error message.
func (e *TemplateError) Error() string {
	return e.message
}
