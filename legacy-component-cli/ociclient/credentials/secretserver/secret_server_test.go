// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secretserver_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/legacy-component-cli/ociclient/credentials/secretserver"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "secretserver Test Suite")
}

var _ = ginkgo.Describe("secret server", func() {

	ginkgo.It("should read from unencrypted server", func() {
		svr := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(200)
			_, _ = writer.Write([]byte(`
{
  "container_registry": {
    "a": {
      "username": "abc"
    }
  }
}
`))
		}))
		defer svr.Close()
		Expect(os.Setenv(secretserver.EndpointEnvVarName, svr.URL)).To(Succeed())
		Expect(os.Unsetenv(secretserver.SecretKeyEnvVarName))

		ss, err := secretserver.NewSecretServer()
		Expect(err).ToNot(HaveOccurred())

		config, err := ss.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(config.ContainerRegistry).To(HaveKey("a"))
	})

	ginkgo.It("should read from encrypted server", func() {
		svr := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer ginkgo.GinkgoRecover()
			writer.WriteHeader(200)
			data, _ := base64.StdEncoding.DecodeString("gqxpvIjAFda7NS3XAVqn/SIVujnGsG0dPjG5s0S5+TyszCVEQ8QF9di+7ZpVpHnCnRXUuZ6HSKV2B1tRi+Lf6MQxr7U5QGix6FB3/8YdYr8=")
			n, err := writer.Write(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(80))
		}))
		defer svr.Close()
		Expect(os.Setenv(secretserver.EndpointEnvVarName, svr.URL)).To(Succeed())
		Expect(os.Setenv(secretserver.ConcourseConfigEnvVarName, "encrypted-concourse-secrets/encrypted_concourse_cfg")).To(Succeed())
		Expect(os.Setenv(secretserver.CipherEnvVarName, "AES.ECB")).To(Succeed())
		// this is a example key only generated for this test
		Expect(os.Setenv(secretserver.SecretKeyEnvVarName, "YWJocmdmb2tkdHdlY2Znbg==")).To(Succeed()) // ggignore

		ss, err := secretserver.NewSecretServer()
		Expect(err).ToNot(HaveOccurred())

		config, err := ss.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(config.ContainerRegistry).To(HaveKey("a"))
	})

})
