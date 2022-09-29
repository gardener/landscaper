// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package reconcilehelper_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

func createDefaultContextsForNamespaces(kubeClient client.Client) {
	// create default repo for all namespaces
	repoCtx := componentsregistry.NewLocalRepository("../testdata/registry")
	for i := 1; i <= 11; i++ {
		Expect(testutils.CreateDefaultContext(context.TODO(), kubeClient, repoCtx, fmt.Sprintf("test%d", i))).To(Succeed())
	}
}
