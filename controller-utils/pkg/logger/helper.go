// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logger

import "fmt"

func Logf(logFunc func(msg string, keysAndValues ...interface{}), format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	logFunc(message)
}
