// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

var (
	errorLineColumnRegexp = regexp.MustCompile("(?m):([0-9]+)(:([0-9]+))?:")
)

// TemplateError wraps a go templating error and adds more human-readable information.
type TemplateError struct {
	err            error
	source         *string
	input          map[string]interface{}
	inputFormatter *template.TemplateInputFormatter
	message        string
}

// TemplateErrorBuilder creates a new TemplateError.
func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err:     err,
		message: err.Error(),
	}
}

// WithSource adds the template source code to the error.
func (e *TemplateError) WithSource(source *string) *TemplateError {
	e.source = source
	return e
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

	if e.source != nil {
		builder.WriteString("\ntemplate source:\n")
		builder.WriteString(e.formatSource())
	}

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

// formatSource extracts the significant template source code that was the reason of the template error.
func (e *TemplateError) formatSource() string {
	var (
		err                    error
		errorLine, errorColumn int
	)

	errStr := e.err.Error()
	formatted := strings.Builder{}

	// parse error line and column
	m := errorLineColumnRegexp.FindStringSubmatch(errStr)
	if m == nil {
		return ""
	}

	if len(m) >= 2 {
		// error line
		errorLine, err = strconv.Atoi(m[1])
		if err != nil {
			return ""
		}
	}
	if len(m) >= 4 {
		// error column
		errorColumn, err = strconv.Atoi(m[3])
		if err != nil {
			errorColumn = 0
		}
	}

	formatted.WriteString(CreateSourceSnippet(errorLine, errorColumn, strings.Split(*e.source, "\n")))
	return formatted.String()
}
