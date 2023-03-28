// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions_test

import (
	"context"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/container"
	"github.com/gardener/landscaper/pkg/components/oci"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/executions"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Execution Operation", func() {
	var (
		componentResolver componentsregistry.TypedRegistry
		state             *envtest.State
		kClient           client.Client
		testInstallations map[string]*lsv1alpha1.Installation
		cd                *cdv2.ComponentDescriptor
	)

	BeforeEach(func() {
		var err error
		ctx := context.Background()
		defer ctx.Done()

		state, err = testenv.InitResources(ctx, "./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		kClient = testenv.Client
		testInstallations = state.Installations

		componentResolver, err = componentsregistry.NewLocalClient("./testdata/registry")
		Expect(err).ToNot(HaveOccurred())
		repository := componentsregistry.NewLocalRepository("./testdata/registry")

		cd, err = componentResolver.Resolve(ctx, repository, "example.com/root", "v1.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(cd).ToNot(BeNil())
	})

	It("to create an execution with the correct configuration", func() {
		ctx := context.Background()
		defer ctx.Done()

		var err error

		repoCtx := &cdv2.OCIRegistryRepository{
			ObjectType: cdv2.ObjectType{
				Type: componentResolver.Type(),
			},
			BaseURL: "./testdata/registry",
		}

		var repositoryContext cdv2.UnstructuredTypedObject

		repositoryContext.ObjectType = repoCtx.ObjectType
		repositoryContext.Raw, err = json.Marshal(repoCtx)
		Expect(err).ToNot(HaveOccurred())
		err = json.Unmarshal(repositoryContext.Raw, &repositoryContext.Object)
		Expect(err).ToNot(HaveOccurred())

		lsCtx := &installations.Scope{
			Name:     "default",
			Parent:   nil,
			Siblings: nil,
			External: installations.ExternalContext{
				Context: lsv1alpha1.Context{
					RepositoryContext: &repositoryContext,
				},
				ComponentName:    "example.com/root",
				ComponentVersion: "v1.0.0",
			},
		}

		inst := testInstallations["test1/root"]
		Expect(inst).ToNot(BeNil())
		intBlueprint, err := blueprints.Resolve(ctx, componentResolver, lsCtx.External.ComponentDescriptorRef(), inst.Spec.Blueprint)
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

		registry, _ := oci.NewOCIRegistry(componentResolver)
		installationOperation, err := installations.NewOperationBuilder(internalInst).
			Client(kClient).
			ComponentDescriptor(cd).
			ComponentRegistry(registry).
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

		Expect(providerConfig.ComponentDescriptor.Reference.RepositoryContext.Type).To(Equal(componentResolver.Type()))
		Expect(providerConfig.ComponentDescriptor.Reference.RepositoryContext.Object).To(HaveKey("baseUrl"))
		Expect(providerConfig.ComponentDescriptor.Reference.RepositoryContext.Object["baseUrl"]).To(Equal("./testdata/registry"))

		Expect(providerConfig.Image).To(Equal("example.com/image:v1.0.0"))

		importValues := make(map[string]interface{})
		err = json.Unmarshal(providerConfig.ImportValues, &importValues)
		Expect(err).ToNot(HaveOccurred())
		Expect(importValues).To(HaveKey("verbosity"))
		Expect(importValues["verbosity"]).To(BeEquivalentTo(10))
	})
})
