package spiff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type TemplaterError struct {
	templateName string
	inputData map[string]interface{}
	err error
}

func (e *TemplaterError) FormatParseError() error {
	return nil
}

func (e *TemplaterError) FormatExecuteError() error {
	errorMessage := strings.Builder{}
	errorMessage.WriteString(e.err.Error())
	errorMessage.WriteString("\n\ntemplate input:\n")
	errorMessage.WriteString(e.formatInput())
	return fmt.Errorf("%s", errorMessage.String())
}

func sanitize(root map[string]interface{}) {
	for k, v := range root {
		child, ok := v.(map[string]interface{})
		if ok {
			sanitize(child)
		} else {
			root[k] = fmt.Sprintf("[...] (%s)", reflect.TypeOf(v).String())
		}
	}
}

func (e *TemplaterError) formatInput() string {
	if e.inputData == nil {
		return ""
	}

	formatted := strings.Builder{}

	for k, e := range e.inputData {
		if k == "imports" {
			source, ok := e.(map[string]interface{})
			if ok {
				sanitized := make(map[string]interface{})
				for k, v := range source {
					sanitized[k] = v
				}
				sanitize(sanitized)
				e = sanitized
			}
		}

		marshaled, err := json.Marshal(e)

		if err != nil {
			marshaled = []byte("")
		}

		formatted.WriteString(fmt.Sprintf("\t%s: %s\n", k, string(marshaled)))
	}

	return formatted.String()
}
