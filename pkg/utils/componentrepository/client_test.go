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

package componentrepository_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	logtesting "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/landscaper/pkg/utils/componentrepository"
	mock_oci "github.com/gardener/landscaper/pkg/utils/oci/mock"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComponentRepository Test Suite")
}

var _ = Describe("Client", func() {

	var (
		ctrl *gomock.Controller
		ociClient *mock_oci.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		ociClient = mock_oci.NewMockClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should fetch and return a component descriptor when a valid tar is returned", func() {
		cdClient, err := componentrepository.NewWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: componentrepository.ComponentDescriptorMediaType,
			Digest: "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)
		ociClient.EXPECT().Fetch(ctx, "example.com/my-comp:0.0.1", cdLayerDesc, gomock.Any()).Return(nil).Do(func(ctx context.Context, ref string, desc ocispecv1.Descriptor, writer io.Writer){
			data, err := ioutil.ReadFile("./testdata/comp1.tar")
			Expect(err).ToNot(HaveOccurred())
			_, err = io.Copy(writer, bytes.NewBuffer(data))
			Expect(err).ToNot(HaveOccurred())
		})

		_, err = cdClient.Get(ctx, "example.com", ref)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should throw an error if the manifest has more layers", func() {
		cdClient, err := componentrepository.NewWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: componentrepository.ComponentDescriptorMediaType,
			Digest: "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{
				cdLayerDesc,
				{
					Digest: "1.2.3",
				},
			},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Get(ctx, "example.com", ref)
		Expect(err).To(HaveOccurred())
	})

	It("should throw an error if the manifest has a unknown type", func() {
		cdClient, err := componentrepository.NewWithOCIClient(logtesting.NullLogger{}, ociClient)
		Expect(err).ToNot(HaveOccurred())
		ctx := context.Background()
		defer ctx.Done()

		ref := cdv2.ObjectMeta{
			Name:    "my-comp",
			Version: "0.0.1",
		}
		cdLayerDesc := ocispecv1.Descriptor{
			MediaType: "unknown-type",
			Digest: "1.2.3",
		}
		manifest := &ocispecv1.Manifest{
			Layers: []ocispecv1.Descriptor{cdLayerDesc},
		}

		ociClient.EXPECT().GetManifest(ctx, "example.com/my-comp:0.0.1").Return(manifest, nil)

		_, err = cdClient.Get(ctx, "example.com", ref)
		Expect(err).To(HaveOccurred())
	})


})
