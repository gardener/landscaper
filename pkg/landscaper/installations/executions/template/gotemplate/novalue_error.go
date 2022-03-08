// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"fmt"
	"strings"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

const (
	noValue = "<no value>"
)

// NoValueError is used for when an executed template result contains fields with "no value".
type NoValueError struct {
	templateResult string
	templateName   string
	input          map[string]interface{}
	inputFormatter *template.TemplateInputFormatter
	message        string
}

// CreateErrorIfContainsNoValue creates a NoValueError when the template result contains the "no value" string, otherwise nil is returned.
func CreateErrorIfContainsNoValue(templateResult, templateName string, input map[string]interface{}, inputFormatter *template.TemplateInputFormatter) *NoValueError {
	if strings.Contains(templateResult, noValue) {
		err := &NoValueError{
			templateResult: templateResult,
			templateName:   templateName,
			input:          input,
			inputFormatter: inputFormatter,
		}
		err.buildErrorMessage()
		return err
	}
	return nil
}

// Error returns the error message.
func (e *NoValueError) Error() string {
	return e.message
}

// buildErrorMessage creates the error message for this error.
func (e *NoValueError) buildErrorMessage() {
	var (
		builder = strings.Builder{}
	)

	builder.WriteString(fmt.Sprintf("template \"%s\" contains fields with \"no value\":\n", e.templateName))

	lines := strings.Split(e.templateResult, "\n")

	for line, content := range lines {
		line += 1
		column := strings.Index(content, noValue)

		if column > -1 {
			builder.WriteString(fmt.Sprintf("\nline %d:%d\n", line, column))
			builder.WriteString(CreateSourceSnippet(line, column, lines))
		}
	}

	if e.input != nil && e.inputFormatter != nil {
		builder.WriteString("\ntemplate input:\n")
		builder.WriteString(e.inputFormatter.Format(e.input, "\t"))
	}

	e.message = builder.String()
}
