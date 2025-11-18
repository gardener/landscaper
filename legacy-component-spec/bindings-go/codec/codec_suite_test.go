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

package codec_test

import (
	"os"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Utils Test Suite")
}

var _ = ginkgo.Describe("serializer", func() {

	ginkgo.It("should decode a simple component", func() {
		data, err := os.ReadFile("../../language-independent/test-resources/component_descriptor_v2.yaml")
		Expect(err).ToNot(HaveOccurred())

		var comp v2.ComponentDescriptor
		err = codec.Decode(data, &comp)
		Expect(err).ToNot(HaveOccurred())

		Expect(comp.Name).To(Equal("github.com/gardener/gardener"))
		Expect(comp.Version).To(Equal("v1.7.2"))
		Expect(comp.Resources).To(HaveLen(2))

		intDep := comp.Resources[0]
		Expect(intDep.Name).To(Equal("apiserver"))
		Expect(intDep.Version).To(Equal("v1.7.2"))
		Expect(intDep.GetType()).To(Equal(v2.OCIImageType))
		Expect(intDep.Relation).To(Equal(v2.LocalRelation))
		Expect(intDep.Access.GetType()).To(Equal(v2.OCIRegistryType))

		extDep := comp.Resources[1]
		Expect(extDep.GetName()).To(Equal("grafana"))
		Expect(extDep.GetVersion()).To(Equal("7.0.3"))
		Expect(extDep.GetType()).To(Equal(v2.OCIImageType))
		Expect(extDep.Relation).To(Equal(v2.ExternalRelation))

		Expect(extDep.Access.GetType()).To(Equal(v2.OCIRegistryType))
		ociAccess := &v2.OCIRegistryAccess{}
		Expect(v2.NewCodec(nil, nil, nil).Decode(extDep.Access.Raw, ociAccess)).To(Succeed())
		Expect(ociAccess.ImageReference).To(Equal("registry-1.docker.io/grafana/grafana/7.0.3"))
	})

	ginkgo.It("should encode a simple component", func() {
		data, err := os.ReadFile("../../language-independent/test-resources/component_descriptor_v2.yaml")
		Expect(err).ToNot(HaveOccurred())

		var comp v2.ComponentDescriptor
		err = codec.Decode(data, &comp)
		Expect(err).ToNot(HaveOccurred())

		data, err = codec.Encode(&comp)
		Expect(err).ToNot(HaveOccurred())

		err = codec.Decode(data, &comp)
		Expect(err).ToNot(HaveOccurred())
	})

})
