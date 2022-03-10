// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"fmt"
	"strings"
)

const (
	// sourceCodePrepend the number of lines before the error line that are printed.
	sourceCodePrepend = 5
	// sourceCodeAppend the number of lines after the error line that are printed.
	sourceCodeAppend = 5
)

// CreateSourceSnippet creates an excerpt of lines of source code, containing some lines before
// and after the error line.
// The error line and column will be highlighted and looks like this:
// 14:     updateStrategy: patch
// 15:
// 16:     name: test
// 17:     namespace: {{ .imports.namaspace }}
//                              ˆ≈≈≈≈≈≈≈
// 18:
// 19:     exportsFromManifests:
// 20:     - key: ingressClass
func CreateSourceSnippet(errorLine, errorColumn int, source []string) string {
	var (
		sourceStartLine, sourceEndLine int
		formatted                      = strings.Builder{}
	)

	// convert to zero base index
	errorLine -= 1

	// calculate the starting line of the source code
	sourceStartLine = errorLine - sourceCodePrepend
	if sourceStartLine < 0 {
		sourceStartLine = 0

	}

	errorLine -= sourceStartLine
	source = source[sourceStartLine:]

	// calculate the ending line of the source code
	sourceEndLine = errorLine + sourceCodeAppend + 1
	if sourceEndLine > len(source) {
		sourceEndLine = len(source)
	}

	source = source[:sourceEndLine]

	for i, line := range source {
		// for printing, the line has to be converted back to one based index
		realLine := sourceStartLine + i + 1
		// the prefix contains the line number and some amount of whitespaces to keep the correct indentation
		prefix := fmt.Sprintf("%d:%s", realLine, strings.Repeat(" ", 4-(realLine/10)))
		formatted.WriteString(fmt.Sprintf("%s%s\n", prefix, line))

		if i == errorLine {
			formatted.WriteString(fmt.Sprintf("%s\u02c6≈≈≈≈≈≈≈\n", strings.Repeat(" ", errorColumn+len(prefix))))
		}
	}

	return formatted.String()
}
