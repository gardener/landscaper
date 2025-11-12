// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package process_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process"
	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/process/processors"
)

var _ = Describe("pipeline", func() {

	Context("Process", func() {

		It("should correctly process resource", func() {
			res := cdv2.Resource{
				IdentityObjectMeta: cdv2.IdentityObjectMeta{
					Name:    "my-res",
					Version: "v0.1.0",
					Type:    "ociImage",
				},
			}

			l1 := cdv2.Label{
				Name:  "processor-0",
				Value: json.RawMessage(`"true"`),
			}
			l2 := cdv2.Label{
				Name:  "processor-1",
				Value: json.RawMessage(`"true"`),
			}
			expectedRes := res
			expectedRes.Labels = append(expectedRes.Labels, l1)
			expectedRes.Labels = append(expectedRes.Labels, l2)

			cd := cdv2.ComponentDescriptor{
				ComponentSpec: cdv2.ComponentSpec{
					Resources: []cdv2.Resource{
						res,
					},
				},
			}

			p1 := processors.NewResourceLabeler(l1)
			p2 := processors.NewResourceLabeler(l2)
			pipeline := process.NewResourceProcessingPipeline(p1, p2)

			actualCD, actualRes, err := pipeline.Process(context.TODO(), cd, res)
			Expect(err).ToNot(HaveOccurred())

			Expect(*actualCD).To(Equal(cd))
			Expect(actualRes).To(Equal(expectedRes))
		})

	})
})
