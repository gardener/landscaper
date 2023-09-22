// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-test/deep"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	"github.com/open-component-model/ocm/pkg/runtime"
)

// YAMLEqual compares two yaml structures.
// If value mappings are given, the expected string is evaluated by envsubst, first.
// It is an error for actual to be nil.  Use BeNil() instead.
func YAMLEqual(expected interface{}, subst ...map[string]string) types.GomegaMatcher {
	data, err := AsStructure(expected, subst...)
	if err != nil {
		return &reportError{err}
	}

	return &YAMLEqualMatcher{
		Expected: data,
	}
}

type YAMLEqualMatcher struct {
	Expected interface{}
}

func (matcher *YAMLEqualMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <string>.")
	}

	data, err := AsStructure(actual)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(data, matcher.Expected), nil
}

func (matcher *YAMLEqualMatcher) FailureMessage(actual interface{}) (message string) {
	data, err := AsStructure(actual)
	if err == nil {
		diff := deep.Equal(data, matcher.Expected)
		if len(diff) > 0 {
			eff, _ := runtime.DefaultYAMLEncoding.Marshal(data)
			return fmt.Sprintf(
				"Found\n%s\n%s",
				string(eff),
				fmt.Sprintf("unexpected diff in YAML: \n    %s\n", strings.Join(diff, "\n    ")))
		}
		return "identical"
	}
	return format.Message(actual, "to equal", matcher.Expected, err.Error())
}

func (matcher *YAMLEqualMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to equal", matcher.Expected)
}
