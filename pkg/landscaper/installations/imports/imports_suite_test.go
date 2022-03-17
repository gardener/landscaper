// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"

	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

func createDefaultContextsForNamespace(kubeClient client.Client) {
	// create default repo for all namespaces
	repoCtx := componentsregistry.NewLocalRepository("../testdata/registry")
	list, err := os.ReadDir("./testdata/state")
	Expect(err).To(Succeed())
	for _, d := range list {
		Expect(testutils.CreateDefaultContext(context.TODO(), kubeClient, repoCtx, filepath.Base(d.Name()))).To(Succeed())
	}
}
