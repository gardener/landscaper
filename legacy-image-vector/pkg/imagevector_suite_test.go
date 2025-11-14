// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package pkg_test

import (
	"os"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/legacy-component-spec/bindings-go/codec"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "imagevector Test Suite")
}

func readComponentDescriptor(path string) *cdv2.ComponentDescriptor {
	data, err := os.ReadFile(path)
	Expect(err).ToNot(HaveOccurred())

	cd := &cdv2.ComponentDescriptor{}
	Expect(codec.Decode(data, cd)).ToNot(HaveOccurred())
	return cd
}

func readComponentDescriptors(paths ...string) *cdv2.ComponentDescriptorList {
	list := &cdv2.ComponentDescriptorList{}

	for _, path := range paths {
		cd := readComponentDescriptor(path)
		list.Components = append(list.Components, *cd)
	}
	return list
}
