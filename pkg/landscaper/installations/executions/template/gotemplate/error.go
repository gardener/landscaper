package gotemplate

import (
	"fmt"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template"
	"regexp"
	"strconv"
	"strings"
)

type TemplateError struct {
	err error
	source *string
	inputFormatter *template.TemplateInputFormatter

	message string
}

func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err: err,
	}
}

func (e *TemplateError) WithSource(source *string) *TemplateError {
	e.source = source
	return e
}

func (e *TemplateError) WithInputFormatter(inputFormatter *template.TemplateInputFormatter) *TemplateError {
	e.inputFormatter = inputFormatter
	return e
}

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

func (e *TemplateError) Error() string {
	return e.message
}

func (e *TemplateError) formatSource() string {
	errStr := e.err.Error()
	formatted := strings.Builder{}

	r, err := regexp.Compile("(?m):([0-9]+)(:([0-9]+))?:")
	if err != nil {
		return ""
	}

	errorLine := 0
	errorColumn := -1
	start := 0
	end := 0

	m := r.FindStringSubmatch(errStr)
	if m == nil {
		return ""
	}

	if len(m) >= 2 {
		errorLine, err = strconv.Atoi(m[1])
		if err != nil {
			return ""
		}
		errorLine -= 1
	}
	if len(m) >= 4 {
		errorColumn, err = strconv.Atoi(m[3])
		if err != nil {
			return ""
		}
	}

	lines := strings.Split(*e.source, "\n")

	start = errorLine - 5
	if start < 0 {
		start = 0
	}

	end = errorLine + 5
	if end >= len(lines) {
		end = len(lines) - 1
	}

	lines = lines[start:]

	for i, line := range lines {
		i += start

		if i > end {
			break
		}

		prefix := fmt.Sprintf("%d: ", i)
		formatted.WriteString(fmt.Sprintf("%s%s\n", prefix, line))

		if i == errorLine && errorColumn >= 0 {
			formatted.WriteString(fmt.Sprintf("%s\u02c6≈≈≈≈≈≈≈\n", strings.Repeat(" ", errorColumn + len(prefix))))
		}
	}

	return formatted.String()
}
