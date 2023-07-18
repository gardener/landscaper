// SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var _ logr.LogSink = KeyConflictPreventionLayer{}

const conflictModifierFormatString = "%s_conflict(%d)"

// KeyConflictPreventionLayer is a helper struct. It implements logr.LogSink by containing a LogSink internally,
// to which all method calls are forwarded. The only purpose of this struct is to detect duplicate keys for logr.WithValues
// and replace them to avoid conflicts.
type KeyConflictPreventionLayer struct {
	logr.LogSink
	keys sets.String //nolint:staticcheck // Ignore SA1019 // TODO: change to generic set
}

// PreventKeyConflicts takes a logr.Logger and wraps a KeyConflictPreventionLayer around its LogSink.
// It is already used by the logging framework's constructors and will likely not have to be called from outside the package.
// Mainly exported for testing purposes.
func PreventKeyConflicts(log logr.Logger) logr.Logger {
	return logr.New(KeyConflictPreventionLayer{
		LogSink: log.GetSink(),
		keys:    sets.NewString(),
	})
}
func (kcpl KeyConflictPreventionLayer) wrapKeyConflictLayer(sink logr.LogSink) logr.LogSink {
	return KeyConflictPreventionLayer{
		LogSink: sink,
		keys:    sets.NewString(kcpl.keys.UnsortedList()...),
	}
}

func (kcpl KeyConflictPreventionLayer) Info(level int, msg string, keysAndValues ...interface{}) {
	kcpl.WithValues(keysAndValues...).(KeyConflictPreventionLayer).LogSink.Info(level, msg)
}

func (kcpl KeyConflictPreventionLayer) Error(err error, msg string, keysAndValues ...interface{}) {
	kcpl.WithValues(keysAndValues...).(KeyConflictPreventionLayer).LogSink.Error(err, msg)
}

// WithValues works as usual, but it will replace keys which already exist with a suffixed version indicating the conflict.
func (kcpl KeyConflictPreventionLayer) WithValues(keysAndValues ...interface{}) logr.LogSink {
	var newKeysAndValues []interface{} // lazy copying - if the slice needs to be changed, we have to copy it
	finalKeysAndValues := keysAndValues
	keyset := sets.NewString(kcpl.keys.UnsortedList()...)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, isString := keysAndValues[i].(string)
		if !isString {
			// non-string keys cannot be checked
			continue
		}
		suffixCount := 1
		newKey := key
		for keyset.Has(newKey) {
			newKey = fmt.Sprintf(conflictModifierFormatString, key, suffixCount)
			suffixCount++
		}
		if newKey != key {
			if len(newKeysAndValues) == 0 {
				// initialize copy slice
				newKeysAndValues = make([]interface{}, len(keysAndValues))
				copy(newKeysAndValues, keysAndValues)
				finalKeysAndValues = newKeysAndValues
			}
			newKeysAndValues[i] = newKey
		}
		keyset.Insert(newKey)
	}
	return KeyConflictPreventionLayer{
		LogSink: kcpl.LogSink.WithValues(finalKeysAndValues...),
		keys:    keyset,
	}
}

func (kcpl KeyConflictPreventionLayer) WithName(name string) logr.LogSink {
	return kcpl.wrapKeyConflictLayer(kcpl.LogSink.WithName(name))
}
