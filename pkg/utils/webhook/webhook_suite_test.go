// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package webhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
	// TODO add webhook tests, once kubebuilder has a new release
	// reasoning: latest stable release 2.3.1 doesn't support AdmissionReview version v1
	// it is supported in current alpha releases, but they don't contain the asset binaries (kube-apiserver, etcd, kubectl) which are cumbersome to gather manually
	// therefore we wait for the next stable release and hope they are included there
}
