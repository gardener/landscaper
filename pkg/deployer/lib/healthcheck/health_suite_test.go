// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package healthcheck_test

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/utils/envtest"
)

func TestHealth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Health Suite")
}

var (
	ctx         context.Context
	testenv     *envtest.Environment
	state       *envtest.State
	projectRoot = filepath.Join("..", "..", "..", "..")
	testdataDir = filepath.Join(".", "testdata")

	objectRefs []lsv1alpha1.TypedObjectReference
)

var _ = BeforeSuite(func() {
	var err error
	ctx = context.Background()

	testenv, err = envtest.New(projectRoot)
	Expect(err).ToNot(HaveOccurred())
	_, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())

	state, err = testenv.InitState(ctx)
	Expect(err).ToNot(HaveOccurred())

	objectRefs, err = loadObjectsIntoTestEnv(testdataDir, testenv.Client, state)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})

func loadObjectsIntoTestEnv(dirname string, c client.Client, s *envtest.State) ([]lsv1alpha1.TypedObjectReference, error) {
	decoder := serializer.NewCodecFactory(c.Scheme()).UniversalDecoder()
	files, err := ioutil.ReadDir(dirname)
	Expect(err).ToNot(HaveOccurred())

	objectRefs := make([]lsv1alpha1.TypedObjectReference, len(files))
	for i, n := range files {
		raw, err := ioutil.ReadFile(filepath.Join(dirname, n.Name()))
		if err != nil {
			return nil, err
		}

		u := &unstructured.Unstructured{}
		_, _, err = decoder.Decode(raw, nil, u)
		if err != nil {
			return nil, err
		}

		u.SetNamespace(s.Namespace)
		err = s.Create(ctx, c, u)
		if err != nil {
			return nil, err
		}

		ref := kutil.TypedObjectReferenceFromUnstructuredObject(u)
		objectRefs[i] = *ref
	}

	return objectRefs, nil
}
