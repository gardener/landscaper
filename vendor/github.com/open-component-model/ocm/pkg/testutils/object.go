// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
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
)

// DeepEqual compares two objects and shows diff on failure.
func DeepEqual(expected interface{}) types.GomegaMatcher {
	return &DeepEqualMatcher{
		Expected: expected,
	}
}

type DeepEqualMatcher struct {
	Expected interface{}
}

func (matcher *DeepEqualMatcher) Match(actual interface{}) (success bool, err error) {
	return reflect.DeepEqual(actual, matcher.Expected), nil
}

func (matcher *DeepEqualMatcher) FailureMessage(actual interface{}) (message string) {
	diff := deep.Equal(actual, matcher.Expected)
	if len(diff) > 0 {
		return fmt.Sprintf("unexpected diff in deep equal: \n    %s\n", strings.Join(diff, "\n    "))
	}
	return format.Message(actual, "to equal", matcher.Expected)
}

func (matcher *DeepEqualMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to equal", matcher.Expected)
}
