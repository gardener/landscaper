/*
 * Copyright 2023 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package logrusfmt

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/modern-go/reflect2"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
)

type TextFmtFormatter struct {
	TextFormatter
}

func (f *TextFmtFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// massage entry before passing to original formatter
	e := *entry

	e.Message, e.Data = subst(e.Message, e.Data)
	return f.TextFormatter.Format(&e)
}

func subst(msg string, values map[string]interface{}) (string, map[string]interface{}) {
	found := map[string]struct{}{}

	tagFunc := func(w io.Writer, tag string) (int, error) {

		v, ok := values[tag]
		if !ok {
			return w.Write([]byte("{{" + tag + "}}"))
		}
		found[tag] = struct{}{}
		if reflect2.IsNil(v) {
			return 0, nil
		}
		switch v.(type) {
		case string, bool:
		case int, int64, int32, int16, int8:
		case float32, float64:
		case []byte:
		default:
			data, err := json.Marshal(v)
			if err == nil {
				v = string(data)
			}
		}
		if s, ok := v.(string); ok {
			return w.Write([]byte(s))
		}
		return w.Write([]byte(fmt.Sprintf("%#v", v)))
	}
	result := fasttemplate.ExecuteFuncString(msg, "{{", "}}", tagFunc)
	if len(found) > 0 {
		mod := map[string]interface{}{}
		for k, v := range values {
			if _, ok := found[k]; !ok {
				mod[k] = v
			}
		}
		return result, mod
	}
	return result, values
}
