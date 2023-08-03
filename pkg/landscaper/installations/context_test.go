// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"

	"github.com/gardener/landscaper/apis/config"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/components/cnudie/componentresolvers"

	"github.com/gardener/landscaper/pkg/components/model/componentoverwrites"
	"github.com/gardener/landscaper/pkg/components/registries"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	testutils "github.com/gardener/landscaper/test/utils"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Context", func() {

	var (
		op *lsoperation.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		state             *envtest.State
	)

	BeforeEach(func() {
		var err error
		state, err = testenv.InitResources(context.TODO(), "./testdata/state")
		Expect(err).ToNot(HaveOccurred())
		fakeInstallations = state.Installations
		Expect(testutils.CreateExampleDefaultContext(context.TODO(), testenv.Client, "test1", "test2", "test3", "test4")).To(Succeed())

		fakeClient = testenv.Client

		localregistryconfig := &config.LocalRegistryConfiguration{RootPath: "./testdata/registry"}
		registryAccess, err := registries.GetFactory().NewRegistryAccess(context.Background(), nil, nil, nil, localregistryconfig, nil, nil)
		Expect(err).ToNot(HaveOccurred())
		op = lsoperation.NewOperation(fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).SetComponentsRegistry(registryAccess)
	})

	AfterEach(func() {
		Expect(testenv.CleanupState(context.TODO(), state)).To(Succeed())
	})

	It("should show no parent nor siblings for the test1 root", func() {
		ctx := context.Background()

		instRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, instRoot, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(0))
	})

	It("should show no parent and one sibling for the test2/a installation", func() {
		ctx := context.Background()

		inst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/a"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).To(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(1))
	})

	It("should correctly determine the visible context of a installation with its parent and sibling installations", func() {
		ctx := context.Background()

		inst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/b"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, nil)
		Expect(err).ToNot(HaveOccurred())
		lCtx := instOp.Context()

		Expect(lCtx.Parent).ToNot(BeNil())
		Expect(lCtx.Siblings).To(HaveLen(3))

		Expect(lCtx.Parent.GetInstallation().Name).To(Equal("root"))
	})

	It("initialize root installations with default context", func() {
		ctx := context.Background()

		defaultRepoContext, err := componentresolvers.NewLocalRepositoryContext("../testdata/registry")

		Expect(err).ToNot(HaveOccurred())

		inst, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root-test40"])
		Expect(err).ToNot(HaveOccurred())

		instOp, err := installations.NewInstallationOperationFromOperation(ctx, op, inst, &defaultRepoContext)
		Expect(err).ToNot(HaveOccurred())
		repoContextOfOtherRoot := instOp.Context().Siblings[0].GetInstallation().Spec.ComponentDescriptor.Reference.RepositoryContext
		Expect(repoContextOfOtherRoot).ToNot(BeNil())
	})

	Context("GetExternalContext", func() {

		It("should get the reference from the context", func() {
			ctx := context.Background()
			state, err := testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			lsCtx := &lsv1alpha1.Context{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}
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
			ctx := context.Background()
			state, err := testenv.InitState(ctx)
			Expect(err).ToNot(HaveOccurred())

			repoCtx := testutils.DefaultRepositoryContext("test.com")

			lsCtx := &lsv1alpha1.Context{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}
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
			ctx := context.Background()
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
				ctx := context.Background()
				state, err := testenv.InitState(ctx)
				Expect(err).ToNot(HaveOccurred())

				lsCtx := &lsv1alpha1.Context{
					RepositoryContext: testutils.ExampleRepositoryContext(),
				}
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
				ctx := context.Background()
				state, err := testenv.InitState(ctx)
				Expect(err).ToNot(HaveOccurred())

				lsCtx := &lsv1alpha1.Context{
					RepositoryContext: testutils.ExampleRepositoryContext(),
				}
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

	Context("InjectComponentDescriptorRef", func() {
		It("should inject the component ref", func() {
			extCtx := installations.ExternalContext{
				Context: lsv1alpha1.Context{
					RepositoryContext: testutils.ExampleRepositoryContext(),
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
					RepositoryContext: testutils.ExampleRepositoryContext(),
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
			lsCtx := &lsv1alpha1.Context{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}
			_, err := installations.ApplyComponentOverwrite(context.Background(), nil, nil, lsCtx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.RepositoryContext).To(Equal(testutils.ExampleRepositoryContext()))
		})

		It("should overwrite a repository context", func() {
			repoCtx := testutils.DefaultRepositoryContext("foo.bar")

			ref := &lsv1alpha1.ComponentDescriptorReference{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}
			lsCtx := &lsv1alpha1.Context{
				RepositoryContext: testutils.ExampleRepositoryContext(),
			}

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

			_, err := installations.ApplyComponentOverwrite(context.Background(), nil, ow, lsCtx, ref)
			Expect(err).ToNot(HaveOccurred())
			Expect(ref.RepositoryContext).To(Equal(repoCtx))
			Expect(lsCtx.RepositoryContext).To(Equal(testutils.ExampleRepositoryContext()))
		})
	})

})
