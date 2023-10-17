// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// compressThresholdBytes specifies the threshold in bytes at which the value of an input key are compressed.
	compressThresholdBytes = 512
	// removeValuesMaxDepth is the maximum depth at which values of sensitive input keys are removed.
	// Beyond the maximum depth the values are getting truncated.
	removeValuesMaxDepth = 10
)

func init() {
	gob.Register(map[string]interface{}{})
}

// The TemplateInputFormatter formats the input parameter of a template in a human-readable format.
type TemplateInputFormatter struct {
	prettyPrint   bool
	sensitiveKeys sets.String //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
}

// NewTemplateInputFormatter creates a new template input formatter.
// When prettyPrint is set to true, the json output will be formatted with easier readable indentation.
// The parameter sensitiveKeys can contain template input keys which may contain sensitive data.
// When such a key is encountered during formatting, the values of the respective key will be removed.
func NewTemplateInputFormatter(prettyPrint bool, sensitiveKeys ...string) *TemplateInputFormatter {
	return &TemplateInputFormatter{
		prettyPrint:   prettyPrint,
		sensitiveKeys: sets.NewString(sensitiveKeys...),
	}
}

// Format formats the template input into a string value.
// The given prefix is prepended to each line of the formatted output.
func (f *TemplateInputFormatter) Format(input map[string]interface{}, prefix string) string {
	if input == nil {
		return ""
	}

	var (
		err       error
		marshaled []byte
		formatted strings.Builder
	)

	for k, v := range input {
		// If the current key is contained in the list of sensitive keys, all values in each sub-tree will be removed.
		if f.sensitiveKeys.Has(k) {
			source, ok := v.(map[string]interface{})
			if ok {
				// The map is deep copied so that the original input value is not getting modified.
				v, err = deepCopyMap(source)
				if err != nil {
					v = ""
				}
			}
			v = removeValue(v, 1)
		}

		if f.prettyPrint {
			marshaled, err = json.MarshalIndent(v, prefix, "  ")
		} else {
			marshaled, err = json.Marshal(v)
		}

		if err != nil {
			marshaled = []byte("")
		}

		// compress values if pretty print is not enabled and the length exceeds the threshold.
		if !f.prettyPrint && len(marshaled) > compressThresholdBytes {
			formatted.WriteString(fmt.Sprintf("%s%s: >gzip>base64> %s\n", prefix, k, compressAndEncode(string(marshaled))))
		} else {
			formatted.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, k, string(marshaled)))
		}
	}

	return formatted.String()
}

// removeValue removes the value of the input parameter.
// If the input is a map, all leaf values are removed until a certain depth.
// When the maximum depth is reached, the current leaf of the map will be truncated.
func removeValue(val interface{}, depth uint) interface{} {
	m, ok := val.(map[string]interface{})
	if ok && depth <= removeValuesMaxDepth {
		for k, v := range m {
			m[k] = removeValue(v, depth+1)
		}
	} else {
		if val != nil {
			val = reflect.TypeOf(val).String()
		}
		val = fmt.Sprintf("[...] (%s)", val)
	}

	return val
}

// compressAndEncode compresses the input string with gzip and encodes the output with base64.
func compressAndEncode(input string) string {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write([]byte(input))
	if err != nil {
		return ""
	}

	err = writer.Close()
	if err != nil {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return encoded
}

// deepCopyMap deep copies a map.
func deepCopyMap(in map[string]interface{}) (map[string]interface{}, error) {
	var (
		buf  bytes.Buffer
		copy map[string]interface{}
	)

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(in)
	if err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(&buf)
	err = decoder.Decode(&copy)
	if err != nil {
		return nil, err
	}

	return copy, nil
}
