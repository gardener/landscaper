// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/landscaper/installations/executions/template/gotemplate"
)

var _ = Describe("SourceSnippet", func() {
	It("should format a source snippet with the correct format", func() {
		lines := make([]string, 0, 5)
		for i := 0; i < 50; i++ {
			lines = append(lines, fmt.Sprintf("val%d: %d", i, i))
		}
		snippet := gotemplate.CreateSourceSnippet(len(lines)-1, 7, lines)
		expected := "44:   val43: 43\n45:   val44: 44\n46:   val45: 45\n47:   val46: 46\n48:   val47: 47\n49:   val48: 48\n             \u02c6≈≈≈≈≈≈≈\n50:   val49: 49\n"
		Expect(snippet).To(Equal(expected))
	})
})
