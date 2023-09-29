// Copyright by sirupsen
//
// file taken from https://github.com/sirupsen/logrus
// add the support for additional fixed fields.
// Because of usage of many unecessarily provide fields,
// types and functions, all the stuff has to be copied
// to be extended.
//

package logrusfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
)

type fieldKey string

// FieldMap allows customization of the key names for default fields.
type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

// JSONFormatter formats logs into parsable json
type JSONFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	// The format to use is the same than for time.Format or time.Parse from the standard
	// library.
	// The standard Library already provides a set of predefined format.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// DisableHTMLEscape allows disabling html escaping in output
	DisableHTMLEscape bool

	// DataKey allows users to put all the log entry parameters into a nested dictionary at a given key.
	DataKey string

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &JSONFormatter{
	//   	FieldMap: FieldMap{
	// 		 FieldKeyTime:  "@timestamp",
	// 		 FieldKeyLevel: "@level",
	// 		 FieldKeyMsg:   "@message",
	// 		 FieldKeyFunc:  "@caller",
	//    },
	// }
	FieldMap FieldMap

	// FixedFields can be used to definen a fixed order for dedicated fields.
	// They will be rendered before other fields.
	// If defined, the standard field keys should be added, also.
	// The default order is: FieldKeyTime, FieldKeyLevel, FieldKeyMsg,
	// FieldKeyFunc, FieldKeyFile.
	FixedFields []string

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the json data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from json fields.
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	// PrettyPrint will indent all json logs
	PrettyPrint bool
}

// Format renders a single log entry
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(Fields, len(entry.Data)+len(defaultFixedFields))
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	fieldData := data // the map containing the dynamic data fields
	if f.DataKey != "" {
		data = make(Fields, len(defaultFixedFields))
		data[f.DataKey] = fieldData
	}

	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	var funcVal, fileVal string

	if entry.HasCaller() {
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		} else {
			funcVal = entry.Caller.Function
			fileVal = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		}
	}

	fixedKeys := make([]string, 0, len(defaultFixedFields)+len(data))

	fixed := defaultFixedFields
	if f.FixedFields != nil {
		fixed = f.FixedFields
	}

	for _, field := range fixed {
		effName := f.FieldMap.resolve(fieldKey(field))
		switch field {
		case FieldKeyTime:
			if !f.DisableTimestamp {
				data[effName] = entry.Time.Format(timestampFormat)
				fixedKeys = append(fixedKeys, effName)
			}
		case FieldKeyLevel:
			data[effName] = entry.Level.String()
			fixedKeys = append(fixedKeys, effName)
		case FieldKeyMsg:
			if entry.Message != "" {
				data[effName] = entry.Message
				fixedKeys = append(fixedKeys, effName)
			}
		case FieldKeyFunc:
			if funcVal != "" {
				data[effName] = funcVal
				fixedKeys = append(fixedKeys, effName)
			}
		case FieldKeyFile:
			if fileVal != "" {
				data[effName] = fileVal
				fixedKeys = append(fixedKeys, effName)
			}
		default:
			if d, ok := fieldData[field]; ok {
				delete(fieldData, field)
				data[effName] = d
				fixedKeys = append(fixedKeys, effName)
				for i := 0; i < len(keys); i++ {
					if keys[i] == field {
						keys = append(keys[:i], keys[i+1:]...)
						i--
					}
				}
			}
		}
	}

	if f.DataKey != "" {
		fixedKeys = append(fixedKeys, f.DataKey)
	} else {
		sort.Strings(keys)
		fixedKeys = append(fixedKeys, keys...)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	b.WriteString("{")
	nl := ""

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(!f.DisableHTMLEscape)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
		nl = "\n"
	}

	render := map[string]interface{}{}
	for i, field := range fixedKeys {
		buf.Reset()
		render[field] = data[field]
		if err := encoder.Encode(render); err != nil {
			return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
		}
		delete(render, field)

		data := buf.String()
		for j, c := range data {
			if c == '{' {
				data = data[j+1:]
				break
			}
		}
		for j := len(data) - 1; j >= 0; j-- {
			if data[j] == '}' {
				data = data[:j]
				break
			}
		}
		data = strings.TrimRight(data, "\n")

		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(data)
	}

	b.WriteString(nl + "}" + nl)
	return b.Bytes(), nil
}
