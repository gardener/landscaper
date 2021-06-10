// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package installations_test

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/operation"
	"github.com/gardener/landscaper/pkg/utils"
	testutils "github.com/gardener/landscaper/test/utils"
)

var _ = Describe("Operation", func() {

	var (
		kubeClient client.Client
		op         *installations.Operation
	)

	BeforeEach(func() {
		kubeClient = fake.NewClientBuilder().WithScheme(api.LandscaperScheme).Build()
		commonOp := operation.NewOperation(logr.Discard(), kubeClient, api.LandscaperScheme)
		op = &installations.Operation{
			Inst: &installations.Installation{
				InstallationBase: installations.InstallationBase{Info: &lsv1alpha1.Installation{}},
				Blueprint:        &blueprints.Blueprint{Info: &lsv1alpha1.Blueprint{}},
			},
			Operation: commonOp,
		}
	})

	Context("CreateOrUpdateExports", func() {

		It("should sync a target", func() {
			ctx := context.Background()
			defer ctx.Done()
			target := &lsv1alpha1.Target{}
			target.Annotations = map[string]string{
				"ann": "val1",
			}
			target.Labels = map[string]string{
				"lab": "val2",
			}
			target.Spec.Type = "test-type"
			target.Spec.Configuration = lsv1alpha1.NewAnyJSON([]byte("true"))
			targetObj, err := utils.JSONSerializeToGenericObject(target)
			testutils.ExpectNoError(err)

			op.Inst.Info.Name = "test"
			op.Inst.Info.Namespace = "default"
			op.Inst.Blueprint.Info.Imports = []lsv1alpha1.ImportDefinition{
				{
					FieldValueDefinition: lsv1alpha1.FieldValueDefinition{
						Name:       "my-import",
						TargetType: "test-type",
					},
					Type: lsv1alpha1.ImportTypeTarget,
				},
			}
			op.Inst.Imports = map[string]interface{}{
				"my-import": targetObj,
			}

			testutils.ExpectNoError(op.CreateOrUpdateImports(ctx))

			targetList := &lsv1alpha1.TargetList{}
			testutils.ExpectNoError(kubeClient.List(ctx, targetList))
			Expect(targetList.Items).To(HaveLen(1))
			Expect(targetList.Items[0].Annotations).To(HaveKeyWithValue("ann", "val1"))
			Expect(targetList.Items[0].Labels).To(HaveKeyWithValue("lab", "val2"))
			Expect(targetList.Items[0].Spec.Type).To(Equal(lsv1alpha1.TargetType("test-type")))
			Expect(targetList.Items[0].Spec.Configuration.RawMessage).To(Equal(json.RawMessage("true")))
		})

	})

})
