// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package ctf_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/component-spec-bindings-go/apis/v2"
	"github.com/gardener/landscaper/component-spec-bindings-go/ctf"
)

var _ = Describe("ListResolver", func() {

	It("should resolve a component from a list of one component descriptor", func() {
		cd := cdv2.ComponentDescriptor{}
		cd.Name = "example.com/a"
		cd.Version = "0.0.0"
		repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/registry", ""))
		cd.RepositoryContexts = append(cd.RepositoryContexts, &repoCtx)

		lr, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
			Components: []cdv2.ComponentDescriptor{cd},
		})
		Expect(err).ToNot(HaveOccurred())
		res, err := lr.Resolve(context.TODO(), &repoCtx, "example.com/a", "0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Name).To(Equal("example.com/a"))
	})

	It("should resolve a component from a list of multiple component descriptors", func() {
		cd := cdv2.ComponentDescriptor{}
		cd.Name = "example.com/a"
		cd.Version = "0.0.0"
		repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/registry", ""))
		cd.RepositoryContexts = append(cd.RepositoryContexts, &repoCtx)

		cd2 := cdv2.ComponentDescriptor{}
		cd2.Name = "example.com/a"
		cd2.Version = "0.0.0"
		cd2.Labels = cdv2.Labels{
			{
				Name: "test",
			},
		}
		repoCtx2, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/registry2", ""))
		cd2.RepositoryContexts = append(cd.RepositoryContexts, &repoCtx2)

		lr, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
			Components: []cdv2.ComponentDescriptor{cd, cd2},
		})
		Expect(err).ToNot(HaveOccurred())
		res, err := lr.Resolve(context.TODO(), &repoCtx2, "example.com/a", "0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Name).To(Equal("example.com/a"))
		Expect(res.Labels).To(ContainElement(cdv2.Label{
			Name: "test",
		}))

		res, err = lr.Resolve(context.TODO(), &repoCtx, "example.com/a", "0.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Name).To(Equal("example.com/a"))
		Expect(res.Labels).ToNot(ContainElement(cdv2.Label{
			Name: "test",
		}))
	})

	It("should not resolve a component if the repository contexts do not match", func() {
		cd := cdv2.ComponentDescriptor{}
		cd.Name = "example.com/a"
		cd.Version = "0.0.0"
		repoCtx, _ := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("example.com/registry", ""))
		cd.RepositoryContexts = append(cd.RepositoryContexts, &repoCtx)

		lr, err := ctf.NewListResolver(&cdv2.ComponentDescriptorList{
			Components: []cdv2.ComponentDescriptor{cd},
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = lr.Resolve(context.TODO(), &repoCtx, "example.com/b", "0.0.0")
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(ctf.NotFoundError))
	})

})
