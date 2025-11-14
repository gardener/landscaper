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

package v2_test

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
)

var _ = ginkgo.Describe("helper", func() {

	ginkgo.It("should inject a new repository context if none is defined", func() {
		cd := &v2.ComponentDescriptor{}
		Expect(v2.DefaultComponent(cd)).To(Succeed())

		repoCtx := v2.NewOCIRegistryRepository("example.com", "")
		Expect(v2.InjectRepositoryContext(cd, repoCtx)).To(Succeed())
		Expect(cd.RepositoryContexts).To(HaveLen(1))

		Expect(v2.InjectRepositoryContext(cd, repoCtx)).To(Succeed())
		Expect(cd.RepositoryContexts).To(HaveLen(1))

		repoCtx2 := v2.NewOCIRegistryRepository("example.com/dev", "")
		Expect(v2.InjectRepositoryContext(cd, repoCtx2)).To(Succeed())
		Expect(cd.RepositoryContexts).To(HaveLen(2))
	})

})
