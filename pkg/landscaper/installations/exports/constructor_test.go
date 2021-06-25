// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package exports_test

import (
	"context"
	"encoding/json"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/exports"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	kutil "github.com/gardener/landscaper/pkg/utils/kubernetes"
	"github.com/gardener/landscaper/test/utils/envtest"
)

var _ = Describe("Constructor", func() {

	var (
		op *installations.Operation

		fakeInstallations map[string]*lsv1alpha1.Installation
		fakeClient        client.Client
		fakeCompRepo      ctf.ComponentResolver
	)

	BeforeEach(func() {
		var (
			err   error
			state *envtest.State
		)
		fakeClient, state, err = envtest.NewFakeClientFromPath("./testdata/state")
		Expect(err).ToNot(HaveOccurred())

		fakeInstallations = state.Installations

		fakeCompRepo, err = componentsregistry.NewLocalClient(logr.Discard(), "../testdata/registry")
		Expect(err).ToNot(HaveOccurred())

		op = &installations.Operation{
			Operation: lsoperation.NewOperation(logr.Discard(), fakeClient, api.LandscaperScheme, record.NewFakeRecorder(1024)).
				SetComponentsRegistry(fakeCompRepo),
		}
	})

	It("should construct the exported config from its execution", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test2/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot

		c := exports.NewConstructor(op)
		res, _, err := c.Construct(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())
		Expect(res).To(HaveLen(2), "should export 2 data object for 2 exports")

		id := func(element interface{}) string {
			return element.(*dataobjects.DataObject).Metadata.Key
		}
		Expect(res).To(MatchAllElements(id, Elements{
			"root.y": PointTo(MatchFields(IgnoreExtras, Fields{
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
				}),
				"Data": Equal("val-exec-y"),
			})),
			"root.z": PointTo(MatchFields(IgnoreExtras, Fields{
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
				}),
				"Data": Equal("val-exec-z"),
			})),
		}))
	})

	It("should construct the exported config from a child", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		op.Inst.Blueprint.Info.ExportExecutions = []lsv1alpha1.TemplateExecutor{
			{
				Type:     lsv1alpha1.GOTemplateType,
				Template: lsv1alpha1.AnyJSON{RawMessage: []byte(`"exports:\n  root.y: {{ index .values.dataobjects \"root.y\" }}\n  root.z: {{ index .values.dataobjects \"root.z\" }}"`)},
			},
		}

		c := exports.NewConstructor(op)
		res, _, err := c.Construct(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())
		Expect(res).To(HaveLen(2), "should export 2 data object from b and c")

		id := func(element interface{}) string {
			do := element.(*dataobjects.DataObject)
			return do.Metadata.Key
		}
		Expect(res).To(MatchAllElements(id, Elements{
			"root.z": PointTo(MatchFields(IgnoreExtras, Fields{
				"Data": Equal("val-b"),
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
					"Key":        Equal("root.z"),
				}),
			})),
			"root.y": PointTo(MatchFields(IgnoreExtras, Fields{
				"Data": Equal("val-c"),
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
					"Key":        Equal("root.y"),
				}),
			})),
		}))
	})

	It("should forbid the export from a child when the schema is not satisfied", func() {
		ctx := context.Background()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test1/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		op.Inst.Blueprint.Info.ExportExecutions = []lsv1alpha1.TemplateExecutor{
			{
				Type:     lsv1alpha1.GOTemplateType,
				Template: lsv1alpha1.AnyJSON{RawMessage: []byte(`"root.y: true\nroot.z: {{ index .values.dataobjects \"root.z\" }}"`)},
			},
		}

		c := exports.NewConstructor(op)
		_, _, err = c.Construct(ctx)
		Expect(err).To(HaveOccurred())
	})

	It("should construct the exported config from a siblings and the execution config", func() {
		ctx := context.Background()
		defer ctx.Done()
		inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test3/root"])
		Expect(err).ToNot(HaveOccurred())
		op.Inst = inInstRoot
		Expect(op.SetInstallationContext(ctx)).To(Succeed())

		op.Inst.Blueprint.Info.ExportExecutions = []lsv1alpha1.TemplateExecutor{
			{
				Type:     lsv1alpha1.GOTemplateType,
				Template: lsv1alpha1.AnyJSON{RawMessage: []byte(`"exports:\n  root.y: {{ index .values.deployitems.deploy \"root.y\" }}\n  root.z: {{ index .values.dataobjects \"root.z\" }}"`)},
			},
		}

		c := exports.NewConstructor(op)
		res, targets, err := c.Construct(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).ToNot(BeNil())
		Expect(res).To(HaveLen(2), "should export 2 data object from execution and a")
		Expect(targets).To(HaveLen(0))

		id := func(element interface{}) string {
			do := element.(*dataobjects.DataObject)
			return do.Metadata.Key
		}
		Expect(res).To(MatchElements(id, IgnoreExtras, Elements{
			"root.y": PointTo(MatchFields(IgnoreExtras, Fields{
				"Data": Equal("val-exec"),
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
					"Key":        Equal("root.y"),
				}),
			})),
			"root.z": PointTo(MatchFields(IgnoreExtras, Fields{
				"Data": Equal("val-a"),
				"Metadata": MatchFields(IgnoreExtras, Fields{
					"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
					"Key":        Equal("root.z"),
				}),
			})),
		}))
	})

	Context("Target Export", func() {
		It("should export a target from a template and a subinstallation", func() {
			ctx := context.Background()
			defer ctx.Done()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			c := exports.NewConstructor(op)
			_, res, err := c.Construct(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeNil())
			Expect(res).To(HaveLen(2), "should export 2 targets from execution and installation e")

			id := func(element interface{}) string {
				do := element.(*dataobjects.Target)
				return do.Metadata.Key
			}
			Expect(res).To(MatchElements(id, IgnoreExtras, Elements{
				"root.y": PointTo(MatchFields(IgnoreExtras, Fields{
					"Raw": PointTo(MatchFields(IgnoreExtras, Fields{
						"Spec": MatchFields(IgnoreExtras, Fields{
							"Type":          Equal(lsv1alpha1.TargetType("landscaper.gardener.cloud/mock")),
							"Configuration": Equal(lsv1alpha1.AnyJSON{RawMessage: json.RawMessage(`"val-a"`)}),
						}),
					})),
					"Metadata": MatchFields(IgnoreExtras, Fields{
						"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
						"Key":        Equal("root.y"),
					}),
				})),
				"root.z": PointTo(MatchFields(IgnoreExtras, Fields{
					"Raw": PointTo(MatchFields(IgnoreExtras, Fields{
						"Spec": MatchFields(IgnoreExtras, Fields{
							"Type":          Equal(lsv1alpha1.TargetType("landscaper.gardener.cloud/mock")),
							"Configuration": Equal(lsv1alpha1.AnyJSON{RawMessage: json.RawMessage(`"val-e"`)}),
						}),
					})),
					"Metadata": MatchFields(IgnoreExtras, Fields{
						"SourceType": Equal(lsv1alpha1.ExportDataObjectSourceType),
						"Key":        Equal("root.z"),
					}),
				})),
			}))
		})

		It("should forbid export of a wrong target type", func() {
			ctx := context.Background()
			inInstRoot, err := installations.CreateInternalInstallation(ctx, op.ComponentsRegistry(), fakeInstallations["test4/root"])
			Expect(err).ToNot(HaveOccurred())
			op.Inst = inInstRoot
			Expect(op.SetInstallationContext(ctx)).To(Succeed())

			target := &lsv1alpha1.Target{}
			targetName := lsv1alpha1helper.GenerateDataObjectName(lsv1alpha1helper.DataObjectSourceFromInstallation(inInstRoot.Info), "root.z")
			key := kutil.ObjectKey(targetName, "test4")
			Expect(fakeClient.Get(ctx, key, target)).To(Succeed())
			target.Spec.Type = "unknownType"
			Expect(fakeClient.Update(ctx, target))

			c := exports.NewConstructor(op)
			_, _, err = c.Construct(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

})
