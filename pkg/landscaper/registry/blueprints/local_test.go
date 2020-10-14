// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package blueprintsregistry_test

import (
	"context"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr/testing"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	blueprintsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/blueprints"
)

const (
	localTestData1 = "./testdata/local-1"
	localTestData2 = "./testdata/local-2"
)

var _ = Describe("Local Registry", func() {

	var (
		fs vfs.FileSystem
	)

	BeforeEach(func() {
		fs = memoryfs.New()
	})

	Context("initialize Index", func() {
		It("should be successfully initialized with one path", func() {
			_, err := blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, localTestData1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be successfully initialized with multiple paths", func() {
			_, err := blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, localTestData1, localTestData2)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be successfully initialized with multiple paths that are subpaths", func() {
			_, err := blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, localTestData1, fmt.Sprintf("%s/comp1", localTestData1))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("GetBlueprint", func() {

		var reg blueprintsregistry.Registry

		BeforeEach(func() {
			var err error
			reg, err = blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, localTestData1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a component by name", func() {
			_, err := reg.GetBlueprint(context.TODO(), newLocalComponent("root-definition", "1.0.0"))
			Expect(err).ToNot(HaveOccurred())

			_, err = reg.GetBlueprint(context.TODO(), newLocalComponent("sub-definition-1", "1.1.0"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error if the name is incorrect", func() {
			_, err := reg.GetBlueprint(context.TODO(), newLocalComponent("unkown-definition", "1.0.0"))
			Expect(blueprintsregistry.IsComponentNotFoundError(err)).To(BeTrue())
		})

		It("should return an error if the version is incorrect", func() {
			_, err := reg.GetBlueprint(context.TODO(), newLocalComponent("sub-definition-1", "1.0.0"))
			Expect(blueprintsregistry.IsVersionNotFoundError(err)).To(BeTrue())
		})
	})

	Context("GetContent", func() {

		var reg blueprintsregistry.Registry

		BeforeEach(func() {
			var err error
			reg, err = blueprintsregistry.NewLocalRegistry(testing.NullLogger{}, localTestData1)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return the blob for a component by name", func() {
			err := reg.GetContent(context.TODO(), newLocalComponent("root-definition", "1.0.0"), fs)
			Expect(err).ToNot(HaveOccurred())

			fs = memoryfs.New()
			err = reg.GetContent(context.TODO(), newLocalComponent("sub-definition-1", "1.1.0"), fs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be able to list all subcomponents as directories int he blob of the root component", func() {
			err := reg.GetContent(context.TODO(), newLocalComponent("root-definition", "1.0.0"), fs)
			Expect(err).ToNot(HaveOccurred())

			dirs, err := test.List(fs, "/")
			Expect(err).ToNot(HaveOccurred())
			//dirInfo, err := afero.ReadDir(fs, "/")
			//Expect(err).ToNot(HaveOccurred())

			Expect(dirs).To(And(ContainElement("comp1"), ContainElement("comp1-1")))
		})

		It("should be able to read the test file of the subcomponent", func() {
			err := reg.GetContent(context.TODO(), newLocalComponent("sub-definition-1", "1.1.0"), fs)
			Expect(err).ToNot(HaveOccurred())

			file, err := fs.Open("/testdata.txt")
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()
			test.ExpectRead(file, []byte("Test Data"))
		})

		It("should return an error if the name is incorrect", func() {
			err := reg.GetContent(context.TODO(), newLocalComponent("unkown-definition", "1.0.0"), fs)
			Expect(blueprintsregistry.IsComponentNotFoundError(err)).To(BeTrue())
		})

		It("should return an error if the version is incorrect", func() {
			err := reg.GetContent(context.TODO(), newLocalComponent("sub-definition-1", "1.0.0"), fs)
			Expect(blueprintsregistry.IsVersionNotFoundError(err)).To(BeTrue())
		})
	})

})

func newLocalComponent(name, version string) cdv2.Resource {
	return cdv2.Resource{
		ObjectMeta: cdv2.ObjectMeta{
			Name:    name,
			Version: version,
		},
		TypedObjectAccessor: cdv2.NewTypeOnly(lsv1alpha1.BlueprintResourceType),
		Access:              &blueprintsregistry.LocalAccess{ObjectType: cdv2.ObjectType{Type: blueprintsregistry.LocalAccessType}},
	}
}
