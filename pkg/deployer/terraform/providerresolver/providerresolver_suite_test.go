// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package providerresolver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gardener/component-cli/ociclient"
	logtesting "github.com/go-logr/logr/testing"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	terraformv1alpha1 "github.com/gardener/landscaper/apis/deployer/terraform/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/terraform/providerresolver"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ProviderResolver Test Suite")
}

var _ = Describe("ResolveProvider", func() {

	It("should resolve a provider from a public readable url", func() {
		ctx := context.Background()
		defer ctx.Done()
		ociClient, err := ociclient.NewClient(logtesting.NullLogger{})
		Expect(err).ToNot(HaveOccurred())

		memfs := memoryfs.New()
		pluginpath := "/tmp/plugins"
		resolver := providerresolver.NewProviderResolver(logtesting.NullLogger{}, ociClient).
			WithFs(memfs).
			ProvidersDir(pluginpath)
		Expect(resolver.Resolve(ctx, terraformv1alpha1.TerraformProvider{
			Name:    "aws",
			Version: "3.32.0",
			URL:     "https://releases.hashicorp.com/terraform-provider-aws/3.32.0/terraform-provider-aws_3.32.0_linux_amd64.zip",
		})).To(Succeed())

		info, err := memfs.Stat(filepath.Join(pluginpath, "terraform-provider-aws_3.32.0"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Size()).To(BeNumerically(">", 0))
	})

	It("should read a terraform provider from a public readable component descriptor", func() {
		Skip("not implemented yet")
	})

})
