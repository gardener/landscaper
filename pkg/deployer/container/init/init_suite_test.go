// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package init

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	logtesting "github.com/go-logr/logr/testing"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/landscaper/pkg/apis/deployer/container"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installations Imports Test Suite")
}

var _ = Describe("Constructor", func() {

	var (
		fakeClient client.Client
	)

	BeforeEach(func() {
		var (
			err error
		)
		fakeClient, _, err = envtest.NewFakeClientFromPath("./testdata")
		Expect(err).ToNot(HaveOccurred())

		// set default env vars
		Expect(os.Setenv(container.ImportsPathName, container.ImportsPath)).To(Succeed())
		Expect(os.Setenv(container.ExportsPathName, container.ExportsPath)).To(Succeed())
		Expect(os.Setenv(container.StatePathName, container.StatePath)).To(Succeed())
		Expect(os.Setenv(container.ContentPathName, container.ContentPath)).To(Succeed())
	})

	It("should fetch import values from DeployItem and write them to 'import.json'", func() {
		ctx := context.Background()
		defer ctx.Done()
		Expect(os.Setenv(container.DeployItemName, "di-01")).To(Succeed())
		Expect(os.Setenv(container.DeployItemNamespaceName, "default")).To(Succeed())
		opts := &options{}
		opts.Complete(ctx)
		memFs := memoryfs.New()
		Expect(run(ctx, logtesting.NullLogger{}, opts, fakeClient, memFs)).To(Succeed())

		dataBytes, err := vfs.ReadFile(memFs, container.ImportsPath)
		Expect(err).ToNot(HaveOccurred())
		var data interface{}
		Expect(json.Unmarshal(dataBytes, &data)).To(Succeed())
		Expect(data).To(HaveKeyWithValue("key", "val1"))
	})
})
