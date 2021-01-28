// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/onsi/gomega"
)

func ExpectNoError(err error, extra ...interface{}) {
	gomega.ExpectWithOffset(1, err, extra...).ToNot(gomega.HaveOccurred())
}
