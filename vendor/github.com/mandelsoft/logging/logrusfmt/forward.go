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
	"github.com/sirupsen/logrus"
)

type Fields = logrus.Fields

const FieldKeyFile = logrus.FieldKeyFile
const FieldKeyLevel = logrus.FieldKeyLevel
const FieldKeyTime = logrus.FieldKeyTime
const FieldKeyMsg = logrus.FieldKeyMsg
const FieldKeyFunc = logrus.FieldKeyFunc
const FieldKeyLogrusError = logrus.FieldKeyLogrusError

type Entry = logrus.Entry

var AllLevels = logrus.AllLevels

const (
	TraceLevel = logrus.TraceLevel
	DebugLevel = logrus.DebugLevel
	InfoLevel  = logrus.InfoLevel
	WarnLevel  = logrus.WarnLevel
	ErrorLevel = logrus.ErrorLevel
	FatalLevel = logrus.FatalLevel
	PanicLevel = logrus.PanicLevel
)

var defaultFixedFields = []string{
	FieldKeyTime,
	FieldKeyLevel,
	FieldKeyMsg,
	FieldKeyFunc,
	FieldKeyFile,
}

var maxlevellength int

func init() {
	for _, l := range AllLevels {
		if len(l.String()) > maxlevellength {
			maxlevellength = len(l.String())
		}
	}
}
