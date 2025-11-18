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

package ctf_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"

	v2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/ctf"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ctf Test Suite")
}

var _ = ginkgo.Describe("ComponentArchive", func() {

	ginkgo.Context("build", func() {
		ginkgo.It("should build a component archive from path", func() {
			ctx := context.Background()
			defer ctx.Done()
			ca, err := ctf.ComponentArchiveFromPath("./testdata/component-01")
			Expect(err).ToNot(HaveOccurred())
			Expect(ca.ComponentDescriptor.Resources).To(HaveLen(1))

			var data bytes.Buffer
			info, err := ca.Resolve(ctx, ca.ComponentDescriptor.Resources[0], &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.MediaType).To(Equal("json"))
			Expect(data.Bytes()).To(Equal([]byte("{\"some\": \"data\"}")))
		})

		ginkgo.It("should build a component archive from a tar", func() {
			ctx := context.Background()
			defer ctx.Done()
			ca, err := ctf.ComponentArchiveFromPath("./testdata/component-01")
			Expect(err).ToNot(HaveOccurred())

			file, err := os.CreateTemp(os.TempDir(), "ca-tar-")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(os.Remove(file.Name())).ToNot(HaveOccurred())
			}()
			Expect(ca.WriteTar(file)).To(Succeed())

			ca, err = ctf.ComponentArchiveFromCTF(file.Name())
			Expect(err).ToNot(HaveOccurred())

			var data bytes.Buffer
			info, err := ca.Resolve(ctx, ca.ComponentDescriptor.Resources[0], &data)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.MediaType).To(Equal("json"))
			Expect(data.Bytes()).To(Equal([]byte("{\"some\": \"data\"}")))
		})
	})

	ginkgo.It("should build a tar from a component archive", func() {
		ctx := context.Background()
		defer ctx.Done()
		ca, err := ctf.ComponentArchiveFromPath("./testdata/component-01")
		Expect(err).ToNot(HaveOccurred())

		var data bytes.Buffer
		Expect(ca.WriteTar(&data)).To(Succeed())

		fs := memoryfs.New()
		Expect(ctf.ExtractTarToFs(fs, &data)).To(Succeed())

		_, err = fs.Stat(ctf.ComponentDescriptorFileName)
		Expect(err).ToNot(HaveOccurred())
		blobData, err := vfs.ReadFile(fs, "blobs/myblob")
		Expect(err).ToNot(HaveOccurred())
		Expect(blobData).To(Equal([]byte("{\"some\": \"data\"}")))
	})

	ginkgo.It("should add a resource to the component archive from a data reader", func() {
		ctx := context.Background()
		defer ctx.Done()
		ca, err := ctf.ComponentArchiveFromPath("./testdata/component-01")
		Expect(err).ToNot(HaveOccurred())
		Expect(ca.ComponentDescriptor.Resources).To(HaveLen(1))

		data := []byte("test")
		info := &ctf.BlobInfo{
			MediaType: "txt",
			Digest:    digest.FromBytes(data).String(),
			Size:      int64(len(data)),
		}
		res := v2.Resource{
			IdentityObjectMeta: v2.IdentityObjectMeta{
				Name: "res1",
				Type: "txt",
			},
			Relation: v2.ExternalRelation,
		}
		Expect(ca.AddResource(&res, *info, bytes.NewBuffer(data)))
		defer func() {
			Expect(os.Remove(filepath.Join("./testdata/component-01", ctf.BlobPath(info.Digest)))).To(Succeed())
		}()

		Expect(ca.ComponentDescriptor.Resources).To(HaveLen(2))
		var result bytes.Buffer
		info, err = ca.Resolve(ctx, res, &result)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.MediaType).To(Equal("txt"))
		Expect(result.Bytes()).To(Equal(data))
	})

	ginkgo.It("should add a resource to the component archive from a blobresolver", func() {
		ctx := context.Background()
		defer ctx.Done()
		ca, err := ctf.ComponentArchiveFromPath("./testdata/component-01")
		Expect(err).ToNot(HaveOccurred())
		Expect(ca.ComponentDescriptor.Resources).To(HaveLen(1))

		data := []byte("test")
		info := &ctf.BlobInfo{
			MediaType: "txt",
			Digest:    digest.FromBytes(data).String(),
			Size:      int64(len(data)),
		}
		res := v2.Resource{
			IdentityObjectMeta: v2.IdentityObjectMeta{
				Name: "res1",
				Type: "txt",
			},
			Relation: v2.ExternalRelation,
		}
		blobresolver := &testBlobResolver{
			info: func(ctx context.Context, res v2.Resource) (*ctf.BlobInfo, error) { return info, nil },
			resolve: func(ctx context.Context, res v2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
				if _, err := io.Copy(writer, bytes.NewBuffer(data)); err != nil {
					return nil, err
				}
				return info, nil
			},
		}
		Expect(ca.AddResourceFromResolver(ctx, &res, blobresolver))
		defer func() {
			Expect(os.Remove(filepath.Join("./testdata/component-01", ctf.BlobPath(info.Digest)))).To(Succeed())
		}()

		Expect(ca.ComponentDescriptor.Resources).To(HaveLen(2))
		var result bytes.Buffer
		info, err = ca.Resolve(ctx, res, &result)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.MediaType).To(Equal("txt"))
		Expect(result.Bytes()).To(Equal(data))
	})

})

type testBlobResolver struct {
	info    func(ctx context.Context, res v2.Resource) (*ctf.BlobInfo, error)
	resolve func(ctx context.Context, res v2.Resource, writer io.Writer) (*ctf.BlobInfo, error)
}

func (t *testBlobResolver) Info(ctx context.Context, res v2.Resource) (*ctf.BlobInfo, error) {
	return t.info(ctx, res)
}
func (t *testBlobResolver) Resolve(ctx context.Context, res v2.Resource, writer io.Writer) (*ctf.BlobInfo, error) {
	return t.resolve(ctx, res, writer)
}
