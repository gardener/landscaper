// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions_test

import (
	"context"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/apis/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	"github.com/gardener/landscaper/pkg/components/model"
	"github.com/gardener/landscaper/pkg/components/model/types"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Execution Operation", func() {
	var (
		ctx  context.Context
		octx ocm.Context

		registryAccess    model.RegistryAccess
		repositoryContext types.UnstructuredTypedObject
		componentVersion  model.ComponentVersion
		state             *envtest.State
		kClient           client.Client
		testInstallations map[string]*lsv1alpha1.Installation
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		var err error

		state, err = testenv.InitResources(ctx, "./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		kClient = testenv.Client
		testInstallations = state.Installations

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "./testdata/registry/root"}
		registryAccess, err = registries.GetFactory().NewRegistryAccess(ctx, nil, nil, nil,
			localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		Expect(repositoryContext.UnmarshalJSON([]byte(`{"type":"local"}`))).To(Succeed())

		componentVersion, err = registryAccess.GetComponentVersion(ctx, &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repositoryContext,
			ComponentName:     "example.com/root",
			Version:           "v1.0.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(componentVersion).ToNot(BeNil())
	})

	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	It("to create an execution with the correct configuration", func() {
		var err error

		lsCtx := &installations.Scope{
			Name:   "default",
			Parent: nil,
			External: installations.ExternalContext{
				ResultingRepositoryContext: &repositoryContext,
				ComponentName:              "example.com/root",
				ComponentVersion:           "v1.0.0",
			},
		}

		inst := testInstallations["test1/root"]
		Expect(inst).ToNot(BeNil())
		intBlueprint, err := blueprints.Resolve(ctx, registryAccess, lsCtx.External.ComponentDescriptorRef(), inst.Spec.Blueprint, nil)
		Expect(err).ToNot(HaveOccurred())

		internalInst := installations.NewInstallationImportsAndBlueprint(inst, intBlueprint)
		Expect(internalInst).ToNot(BeNil())

		internalInst.SetImports(map[string]interface{}{
			"verbosity": 10,
			"memory": map[string]interface{}{
				"min": 128,
				"max": 1024,
			},
		})

		installationOperation, err := installations.NewOperationBuilder(internalInst).
			WithLsUncachedClient(kClient).
			ComponentVersion(componentVersion).
			ComponentRegistry(registryAccess).
			WithContext(lsCtx).
			Build(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(installationOperation).ToNot(BeNil())

		executionOperation := executions.New(installationOperation)
		Expect(executionOperation).ToNot(BeNil())

		err = executionOperation.Ensure(ctx, internalInst)
		Expect(err).ToNot(HaveOccurred())

		execution := &lsv1alpha1.Execution{}
		err = kClient.Get(ctx, client.ObjectKey{Name: "root", Namespace: "test1"}, execution)
		Expect(err).ToNot(HaveOccurred())
		Expect(execution.Spec.DeployItems).To(HaveLen(1))

		deployItem := execution.Spec.DeployItems[0]
		providerConfig := &container.ProviderConfiguration{}
		err = json.Unmarshal(deployItem.Configuration.Raw, providerConfig)
		Expect(err).ToNot(HaveOccurred())

		Expect(providerConfig.ComponentDescriptor.Reference.ComponentName).To(Equal("example.com/root"))
		Expect(providerConfig.ComponentDescriptor.Reference.Version).To(Equal("v1.0.0"))

		Expect(providerConfig.ComponentDescriptor.Reference.RepositoryContext.Type).To(Equal(repositoryContext.GetType()))

		Expect(providerConfig.Image).To(Equal("example.com/image:v1.0.0"))

		importValues := make(map[string]interface{})
		err = json.Unmarshal(providerConfig.ImportValues, &importValues)
		Expect(err).ToNot(HaveOccurred())
		Expect(importValues).To(HaveKey("verbosity"))
		Expect(importValues["verbosity"]).To(BeEquivalentTo(10))
	})
})
