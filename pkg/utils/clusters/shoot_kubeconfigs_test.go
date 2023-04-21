// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package clusters

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("Shoot Clusters", func() {

	It("should determine the name of a shoot cluster", func() {
		ctx := context.Background()

		const shootName = "testshoot-a"

		kubeconfig := clientcmdapi.Config{
			Kind:       "Config",
			APIVersion: "v1",
			Clusters: map[string]*clientcmdapi.Cluster{
				"cluster-a": {
					Server: fmt.Sprintf("https://api.%s.test.test.test", shootName),
				},
				"cluster-b": {
					Server: "https://api.testshoot-b.test.test.test",
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				"context-a": {
					Cluster: "cluster-a",
				},
				"context-b": {
					Cluster: "cluster-b",
				},
			},
			CurrentContext: "context-a",
		}

		kubeconfigBytes, err := clientcmd.Write(kubeconfig)
		Expect(err).NotTo(HaveOccurred())

		result, err := GetShootClusterNameFromKubeconfig(ctx, kubeconfigBytes)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(shootName))
	})
})
