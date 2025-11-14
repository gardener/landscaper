// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ociclient_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/credentials"
	"github.com/gardener/landscaper/legacy-component-cli/ociclient/test/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ociclient Test Suite")
}

var (
	testenv *envtest.Environment
	client  ociclient.ExtendedClient
	keyring *credentials.GeneralOciKeyring
)

var _ = ginkgo.BeforeSuite(func() {
	testenv = envtest.New(envtest.Options{
		RegistryBinaryPath: filepath.Join("../", envtest.DefaultRegistryBinaryPath),
		Stdout:             ginkgo.GinkgoWriter,
		Stderr:             ginkgo.GinkgoWriter,
	})
	Expect(testenv.Start(context.Background())).To(Succeed())

	keyring = credentials.New()
	Expect(keyring.AddAuthConfig(testenv.Addr, credentials.AuthConfig{
		Username: testenv.BasicAuth.Username,
		Password: testenv.BasicAuth.Password,
	})).To(Succeed())
	var err error
	client, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring))
	Expect(err).ToNot(HaveOccurred())
}, 60)

var _ = ginkgo.AfterSuite(func() {
	Expect(testenv.Close()).To(Succeed())
})
