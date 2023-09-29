// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"fmt"
	"strings"

	"github.com/drone/envsubst"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

// StringEqualTrimmedWithContext compares two trimmed strings and provides the complete actual value
// as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringEqualTrimmedWithContext(expected string, subst ...map[string]string) types.GomegaMatcher {
	var err error
	expected, err = eval(expected, subst...)
	if err != nil {
		return &reportError{err}
	}
	return &StringEqualMatcher{
		Expected: expected,
		Trim:     true,
	}
}

// StringEqualWithContext compares two strings and provides the complete actual value
// as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringEqualWithContext(expected string, subst ...map[string]string) types.GomegaMatcher {
	var err error
	expected, err = eval(expected, subst...)
	if err != nil {
		return &reportError{err}
	}
	return &StringEqualMatcher{
		Expected: expected,
	}
}

type StringEqualMatcher struct {
	Expected string
	Trim     bool
}

func (matcher *StringEqualMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <string>.")
	}

	s, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("Actual value is no string, but a %T.", actual)
	}
	if matcher.Trim {
		return strings.TrimSpace(s) == strings.TrimSpace(matcher.Expected), nil
	}
	return s == matcher.Expected, nil
}

func (matcher *StringEqualMatcher) FailureMessage(actual interface{}) (message string) {
	actualString, actualOK := actual.(string)
	if actualOK {
		compare, expected := actualString, matcher.Expected
		if matcher.Trim {
			compare = strings.TrimSpace(actualString)
			expected = strings.TrimSpace(matcher.Expected)
		}
		return fmt.Sprintf(
			"Found\n%s\n%s",
			actualString,
			format.MessageWithDiff(compare, "to equal", expected),
		)
	}
	return format.Message(actual, "to equal", matcher.Expected)
}

func (matcher *StringEqualMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to equal", matcher.Expected)
}

func eval(expected string, subst ...map[string]string) (string, error) {
	if len(subst) > 0 {
		return envsubst.Eval(expected, stringmapping(subst...))
	}
	return expected, nil
}

func stringmapping(values ...map[string]string) func(variable string) string {
	return func(variable string) string {
		for _, m := range values {
			if v, ok := m[variable]; ok {
				return v
			}
		}
		return "${" + variable + "}"
	}
}

type reportError struct {
	err error
}

func (r *reportError) Match(actual interface{}) (success bool, err error) {
	return false, err
}

func (r *reportError) FailureMessage(actual interface{}) (message string) {
	return r.err.Error()
}

func (r *reportError) NegatedFailureMessage(actual interface{}) (message string) {
	return r.err.Error()
}
