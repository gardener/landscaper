// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gardener/landscaper/pkg/components/cnudie/oci"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutils "github.com/gardener/landscaper/test/utils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

func createDefaultContextsForNamespace(kubeClient client.Client) {
	// create default repo for all namespaces
	repoCtx := oci.NewLocalRepository("../testdata/registry")
	list, err := os.ReadDir("./testdata/state")
	Expect(err).To(Succeed())
	for _, d := range list {
		Expect(testutils.CreateDefaultContext(context.TODO(), kubeClient, repoCtx, filepath.Base(d.Name()))).To(Succeed())
	}
}
