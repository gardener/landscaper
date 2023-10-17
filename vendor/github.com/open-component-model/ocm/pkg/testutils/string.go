// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/drone/envsubst"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	"github.com/open-component-model/ocm/pkg/errors"
)

type Substitutions = map[string]string

func SubstList(values ...string) map[string]string {
	r := map[string]string{}
	for i := 0; i+1 < len(values); i += 2 {
		r[values[i]] = values[i+1]
	}
	return r
}

func SubstFrom(v interface{}, prefix ...string) map[string]string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var values map[string]string
	err = json.Unmarshal(data, &values)
	if err != nil {
		panic(err)
	}
	if len(prefix) > 0 {
		p := strings.Join(prefix, "")
		n := map[string]string{}
		for k, v := range values {
			n[p+k] = v
		}
		values = n
	}
	return values
}

func MergeSubst(subst ...map[string]string) map[string]string {
	r := map[string]string{}
	for _, s := range subst {
		for k, v := range s {
			r[k] = v
		}
	}
	return r
}

// StringEqualTrimmedWithContext compares two trimmed strings and provides the complete actual value
// as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringEqualTrimmedWithContext(expected string, subst ...Substitutions) types.GomegaMatcher {
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

// StringMatchTrimmedWithContext matches a trimmed string by a regular
// expression and provides the complete actual value as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringMatchTrimmedWithContext(expected string, subst ...Substitutions) types.GomegaMatcher {
	var err error
	expected, err = eval(expected, subst...)
	if err != nil {
		return &reportError{err}
	}
	return &StringEqualMatcher{
		Expected: expected,
		Trim:     true,
		Regex:    true,
	}
}

// StringEqualWithContext compares two strings and provides the complete actual value
// as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringEqualWithContext(expected string, subst ...Substitutions) types.GomegaMatcher {
	var err error
	expected, err = eval(expected, subst...)
	if err != nil {
		return &reportError{err}
	}
	return &StringEqualMatcher{
		Expected: expected,
	}
}

// StringMatchWithContext matches a string by a regular expression and provides
// the complete actual value as error context.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func StringMatchWithContext(expected string, subst ...Substitutions) types.GomegaMatcher {
	var err error
	expected, err = eval(expected, subst...)
	if err != nil {
		return &reportError{err}
	}
	return &StringEqualMatcher{
		Expected: expected,
		Regex:    true,
	}
}

type StringEqualMatcher struct {
	Expected string
	Trim     bool
	Regex    bool
}

func (matcher *StringEqualMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <string>.")
	}

	s, err := AsString(actual)
	if err != nil {
		return false, err
	}
	if matcher.Regex {
		expected := matcher.Expected
		if matcher.Trim {
			expected = strings.TrimSpace(expected)
		}
		r, err := regexp.Compile(expected)
		if err != nil {
			return false, errors.Wrapf(err, "Invalid regular expression %q", matcher.Regex)
		}
		if matcher.Trim {
			return r.MatchString(strings.TrimSpace(s)), nil
		}
		return r.MatchString(s), nil
	} else {
		if matcher.Trim {
			return strings.TrimSpace(s) == strings.TrimSpace(matcher.Expected), nil
		}
		return s == matcher.Expected, nil
	}
}

func (matcher *StringEqualMatcher) FailureMessage(actual interface{}) (message string) {
	actualString, err := AsString(actual)
	if err == nil {
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
	actualString, err := AsString(actual)
	if err == nil {
		return format.Message(actualString, "not to equal", matcher.Expected)
	}
	return format.Message(actual, "not to equal", matcher.Expected)
}

func eval(expected string, subst ...Substitutions) (string, error) {
	if len(subst) > 0 {
		return envsubst.Eval(expected, stringmapping(subst...))
	}
	return expected, nil
}

func stringmapping(values ...Substitutions) func(variable string) string {
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
