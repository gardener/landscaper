// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

func createDefaultContextsForNamespace(kubeClient client.Client) {
	// create default repo for all namespaces
	repoCtx := testutils.MakeRepositoryContext(componentsregistry.NewLocalRepository("../testdata/registry"))
	for i := 1; i <= 10; i++ {
		lsCtx := &lsv1alpha1.Context{}
		lsCtx.Name = lsv1alpha1.DefaultContextName
		lsCtx.Namespace = fmt.Sprintf("test%d", i)
		lsCtx.RepositoryContext = repoCtx
		Expect(kubeClient.Create(context.TODO(), lsCtx)).To(Succeed())
	}
}
