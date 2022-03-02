// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
)

var (
	errorLineColumnRegexp = regexp.MustCompile("(?m):([0-9]+)(:([0-9]+))?:")
)

const (
	// sourceCodePrepend the number of lines before the error line that are printed.
	sourceCodePrepend = 5
	// sourceCodeAppend the number of lines after the error line that are printed.
	sourceCodeAppend = 5
)

// TemplateError wraps a go templating error and adds more human-readable information.
type TemplateError struct {
	err            error
	source         *string
	inputFormatter *template.TemplateInputFormatter

	message string
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

// WithInputFormatter adds a template input formatter to the error.
func (e *TemplateError) WithInputFormatter(inputFormatter *template.TemplateInputFormatter) *TemplateError {
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

// formatSource extracts the significant template source code that was the reason of the template error.
// The error line and column will be highlighted and looks like this:
// 14:     updateStrategy: patch
// 15:
// 16:     name: test
// 17:     namespace: {{ .imports.namaspace }}
//                              ˆ≈≈≈≈≈≈≈
// 18:
// 19:     exportsFromManifests:
// 20:     - key: ingressClass
func (e *TemplateError) formatSource() string {
	var (
		err                                                    error
		errorLine, errorColumn, sourceStartLine, sourceEndLine int
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
		errorLine -= 1
	}
	if len(m) >= 4 {
		// error column
		errorColumn, err = strconv.Atoi(m[3])
		if err != nil {
			errorColumn = 0
		}
	}

	lines := strings.Split(*e.source, "\n")

	// calculate the starting line of the source code
	sourceStartLine = errorLine - sourceCodePrepend
	if sourceStartLine < 0 {
		sourceStartLine = 0

	}

	errorLine -= sourceStartLine
	lines = lines[sourceStartLine:]

	// calculate the ending line of the source code
	sourceEndLine = errorLine + sourceCodeAppend + 1
	if sourceEndLine > len(lines) {
		sourceEndLine = len(lines)
	}

	lines = lines[:sourceEndLine]

	for i, line := range lines {
		prefix := fmt.Sprintf("%d: ", i)
		formatted.WriteString(fmt.Sprintf("%s%s\n", prefix, line))

		if i == errorLine {
			formatted.WriteString(fmt.Sprintf("%s\u02c6≈≈≈≈≈≈≈\n", strings.Repeat(" ", errorColumn+len(prefix))))
		}
	}

	return formatted.String()
}
