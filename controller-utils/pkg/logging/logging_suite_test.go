// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging_test

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Test Suite")
}

var _ = Describe("Logging Framework Tests", func() {

	It("should not modify the logger if any method is called", func() {
		compareToLogger := logging.Wrap(logging.PreventKeyConflicts(logr.Discard()))
		log := logging.Wrap(logging.PreventKeyConflicts(logr.Discard()))
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue())

		log.Debug("foo", "bar", "baz", "bar", "baz")
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue(), "calling log.Debug should not modify the logger")

		log.Info("foo", "bar", "baz", "bar", "baz")
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue(), "calling log.Info should not modify the logger")

		log.Error(nil, "foo", "bar", "baz", "bar", "baz")
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue(), "calling log.Error should not modify the logger")

		log.WithName("myname")
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue(), "calling log.WithName should not modify the logger")

		log.WithValues("foo", "bar")
		Expect(reflect.DeepEqual(log, compareToLogger)).To(BeTrue(), "calling log.WithValues should not modify the logger")
	})

})
