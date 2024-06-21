// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	testutils2 "github.com/gardener/landscaper/pkg/components/testutils"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/apis/config"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Context", func() {

	var (
		ctx  context.Context
		octx ocm.Context

		op *lsoperation.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		state             *envtest.State
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)

		var err error
		state, err = testenv.InitResources(ctx, "./testdata/state")
		Expect(err).ToNot(HaveOccurred())
		fakeInstallations = state.Installations
		Expect(testutils.CreateExampleDefaultContext(ctx, testenv.Client, "test1", "test2", "test3", "test4")).To(Succeed())

		fakeClient = testenv.Client

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "./testdata/registry"}
		registryAccess, err := registries.GetFactory().NewRegistryAccess(ctx, nil, nil, nil,
			localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())
		op = lsoperation.NewOperation(api.LandscaperScheme, record.NewFakeRecorder(1024), fakeClient).SetComponentsRegistry(registryAccess)
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(ctx, state)).To(Succeed())
		Expect(octx.Finalize()).To(Succeed())
	})

	It("should show no parent nor siblings for the test1 root", func() {
		instRoot, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test1/root"],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, instRoot, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.GetSiblings(ctx, op.LsUncachedClient())).To(HaveLen(0))
	})

	It("should show no parent and one sibling for the test2/a installation", func() {
		inst, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test2/a"],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.GetSiblings(ctx, op.LsUncachedClient())).To(HaveLen(1))
	})

	It("should correctly determine the visible context of a installation with its parent and sibling installations", func() {
		inst, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test1/b"],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).ToNot(BeNil())
		Expect(lCtx.GetSiblings(ctx, op.LsUncachedClient())).To(HaveLen(3))

		Expect(lCtx.Parent.GetInstallation().Name).To(Equal("root"))
	})

	It("initialize root installations with default context", func() {
		defaultRepoContext, err := testutils2.NewLocalRepositoryContext("../testdata/registry")

		Expect(err).ToNot(HaveOccurred())

		inst, err := installations.CreateInternalInstallationWithContext(ctx, fakeInstallations["test4/root-test40"],
			op.LsUncachedClient(), op.ComponentsRegistry())
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, &defaultRepoContext)
		Expect(err).ToNot(HaveOccurred())
		siblings, err := instOp.Context().GetSiblings(ctx, op.LsUncachedClient())
		Expect(err).ToNot(HaveOccurred())
		repoContextOfOtherRoot := siblings[0].GetInstallation().Spec.ComponentDescriptor.Reference.RepositoryContext
		Expect(repoContextOfOtherRoot).ToNot(BeNil())
	})

	Context("GetExternalContext", func() {

		It("should get the reference from the context", func() {
			state, err := testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			lsCtx := &lsv1alpha1.Context{}
			lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
			lsCtx.Name = "test"
			lsCtx.Namespace = state.Namespace
			Expect(state.Create(ctx, lsCtx)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Namespace = state.Namespace
			inst.Spec.Context = "test"

			extCtx, err := installations.GetExternalContext(ctx, testenv.Client, inst)
			Expect(err).ToNot(HaveOccurred())
			Expect(inst.Spec.ComponentDescriptor).To(BeNil())
			Expect(extCtx.RepositoryContext.Raw).To(MatchJSON(testutils.ExampleRepositoryContext().Raw))
		})

		It("should get the reference from the installation", func() {
			state, err := testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			repoCtx := testutils.DefaultRepositoryContext("test.com")

			lsCtx := &lsv1alpha1.Context{}
			lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
			lsCtx.Name = "test"
			lsCtx.Namespace = state.Namespace
			Expect(state.Create(ctx, lsCtx)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Namespace = state.Namespace
			inst.Spec.Context = "test"
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: repoCtx,
				},
			}

			extCtx, err := installations.GetExternalContext(ctx, testenv.Client, inst)
			Expect(err).ToNot(HaveOccurred())
			Expect(inst.Spec.ComponentDescriptor.Reference.RepositoryContext.Raw).To(MatchJSON(repoCtx.Raw))
			Expect(extCtx.RepositoryContext.Raw).To(MatchJSON(repoCtx.Raw))
		})

		It("should throw an error if a component name and version is defined but no repository context", func() {
			state, err := testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			lsCtx := &lsv1alpha1.Context{}
			lsCtx.Name = "test"
			lsCtx.Namespace = state.Namespace
			Expect(state.Create(ctx, lsCtx)).To(Succeed())

			inst := &lsv1alpha1.Installation{}
			inst.Namespace = state.Namespace
			inst.Spec.Context = "test"
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					ComponentName: "abc",
				},
			}

			_, err = installations.GetExternalContext(ctx, testenv.Client, inst)
			Expect(err).To(HaveOccurred())
		})

		Context("ComponentVersionOverwrite", func() {

			It("should overwrite a repository context", func() {
				state, err := testenv.InitState(ctx)
				Expect(err).ToNot(HaveOccurred())

				lsCtx := &lsv1alpha1.Context{}
				lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
				lsCtx.Name = "test"
				lsCtx.Namespace = state.Namespace
				lsCtx.ComponentVersionOverwritesReference = lsCtx.Name
				Expect(state.Create(ctx, lsCtx)).To(Succeed())

				inst := &lsv1alpha1.Installation{}
				inst.Namespace = state.Namespace
				inst.Spec.Context = "test"
				inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						RepositoryContext: testutils.ExampleRepositoryContext(),
					},
				}

				repoCtx := testutils.DefaultRepositoryContext("test.com")

				// create component version overwrite
				cvo := &lsv1alpha1.ComponentVersionOverwrites{
					Overwrites: lsv1alpha1.ComponentVersionOverwriteList{
						{
							Source: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: inst.Spec.ComponentDescriptor.Reference.RepositoryContext,
							},
							Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: repoCtx,
							},
						},
					},
				}
				cvo.Name = inst.Spec.Context
				cvo.Namespace = state.Namespace
				Expect(state.Create(ctx, cvo)).To(Succeed())

				extCtx, err := installations.GetExternalContext(ctx, testenv.Client, inst)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdv2.UnstructuredTypesEqual(inst.Spec.ComponentDescriptor.Reference.RepositoryContext, repoCtx)).To(BeTrue())
				Expect(cdv2.UnstructuredTypesEqual(extCtx.RepositoryContext, repoCtx)).To(BeTrue())
			})

			It("should overwrite a repository context defined by the external context", func() {
				state, err := testenv.InitState(ctx)
				Expect(err).ToNot(HaveOccurred())

				lsCtx := &lsv1alpha1.Context{}
				lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
				lsCtx.Name = "test"
				lsCtx.Namespace = state.Namespace
				lsCtx.ComponentVersionOverwritesReference = lsCtx.Name
				Expect(state.Create(ctx, lsCtx)).To(Succeed())

				inst := &lsv1alpha1.Installation{}
				inst.Namespace = state.Namespace
				inst.Spec.Context = "test"
				inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
					Reference: &lsv1alpha1.ComponentDescriptorReference{
						ComponentName: "abc",
					},
				}

				repoCtx := testutils.DefaultRepositoryContext("test.com")

				// create component version overwrite
				cvo := &lsv1alpha1.ComponentVersionOverwrites{
					Overwrites: lsv1alpha1.ComponentVersionOverwriteList{
						{
							Source: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: lsCtx.RepositoryContext,
							},
							Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
								RepositoryContext: repoCtx,
							},
						},
					},
				}
				cvo.Name = inst.Spec.Context
				cvo.Namespace = state.Namespace
				Expect(state.Create(ctx, cvo)).To(Succeed())

				extCtx, err := installations.GetExternalContext(ctx, testenv.Client, inst)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdv2.UnstructuredTypesEqual(inst.Spec.ComponentDescriptor.Reference.RepositoryContext, repoCtx)).To(BeTrue())
				Expect(cdv2.UnstructuredTypesEqual(extCtx.RepositoryContext, repoCtx)).To(BeTrue())
			})
		})

	})

})

var _ = Describe("Context", func() {
	var (
		ctx  context.Context
		octx ocm.Context
	)

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
		octx = ocm.New(datacontext.MODE_EXTENDED)
		ctx = octx.BindTo(ctx)
	})
	AfterEach(func() {
		Expect(octx.Finalize()).To(Succeed())
	})

	Context("InjectComponentDescriptorRef", func() {
		It("should inject the component ref", func() {
			extCtx := installations.ExternalContext{
				Context: lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						RepositoryContext: testutils.ExampleRepositoryContext(),
					},
				},
				ComponentName:    "example.com/a",
				ComponentVersion: "0.0.1",
			}

			inst := &lsv1alpha1.Installation{}
			extCtx.InjectComponentDescriptorRef(inst)
			Expect(inst.Spec.ComponentDescriptor).ToNot(BeNil())
			Expect(inst.Spec.ComponentDescriptor.Reference).To(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
				"RepositoryContext": Equal(extCtx.RepositoryContext),
				"ComponentName":     Equal("example.com/a"),
				"Version":           Equal("0.0.1"),
			})))
		})

		It("should overwrite the component ref", func() {
			extCtx := installations.ExternalContext{
				Context: lsv1alpha1.Context{
					ContextConfiguration: lsv1alpha1.ContextConfiguration{
						RepositoryContext: testutils.ExampleRepositoryContext(),
					},
				},
				ComponentName:    "example.com/a",
				ComponentVersion: "0.0.1",
			}

			inst := &lsv1alpha1.Installation{}
			inst.Spec.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					ComponentName: "a",
					Version:       "b",
				},
			}
			extCtx.InjectComponentDescriptorRef(inst)
			Expect(inst.Spec.ComponentDescriptor).ToNot(BeNil())
			Expect(inst.Spec.ComponentDescriptor.Reference).To(gstruct.PointTo(gstruct.MatchAllFields(gstruct.Fields{
				"RepositoryContext": Equal(extCtx.RepositoryContext),
				"ComponentName":     Equal("example.com/a"),
				"Version":           Equal("0.0.1"),
			})))
		})
	})

	Context("ApplyComponentOverwrite", func() {
		It("should default the component descriptor if not defined in the installation", func() {
			ref := &lsv1alpha1.ComponentDescriptorReference{}
			lsCtx := &lsv1alpha1.Context{}
			lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()
			_, err := installations.ApplyComponentOverwrite(ctx, nil, nil, lsCtx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.RepositoryContext).To(Equal(testutils.ExampleRepositoryContext()))
		})

		It("should overwrite a repository context", func() {
			repoCtx := testutils.DefaultRepositoryContext("foo.bar")

			ref := &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}
			lsCtx := &lsv1alpha1.Context{}
			lsCtx.RepositoryContext = testutils.ExampleRepositoryContext()

			ow := componentoverwrites.NewSubstitutions([]lsv1alpha1.ComponentVersionOverwrite{
				{
					Source: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: ref.RepositoryContext,
					},
					Substitution: lsv1alpha1.ComponentVersionOverwriteReference{
						RepositoryContext: repoCtx,
					},
				},
			})

			_, err := installations.ApplyComponentOverwrite(ctx, nil, ow, lsCtx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.RepositoryContext).To(Equal(repoCtx))
			Expect(lsCtx.RepositoryContext).To(Equal(testutils.ExampleRepositoryContext()))
		})
	})

})
