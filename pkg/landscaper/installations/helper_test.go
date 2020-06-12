// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installations

import (
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	lsv1alpha1 "github.com/gardener/landscaper/pkg/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/kubernetes"
)

var _ = g.Describe("helper", func() {

	g.Context("IsRoot", func() {

		g.It("should validate that a installation with a non ComponentInstallation type owner is a root installation", func() {
			inst := &lsv1alpha1.ComponentInstallation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &corev1.Secret{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, scheme.Scheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := IsRootInstallation(inst)
			Expect(isRoot).To(BeTrue())
		})

		g.It("should validate that a installation with a installation owner is not a root installation", func() {
			inst := &lsv1alpha1.ComponentInstallation{}
			inst.Name = "inst"
			inst.Namespace = "default"
			inst.Labels = map[string]string{lsv1alpha1.EncompassedByLabel: "owner"}

			owner := &lsv1alpha1.ComponentInstallation{}
			owner.Name = "owner"
			owner.Namespace = "default"
			err := controllerutil.SetOwnerReference(owner, inst, kubernetes.LandscaperScheme)
			Expect(err).ToNot(HaveOccurred())

			isRoot := IsRootInstallation(inst)
			Expect(isRoot).To(BeFalse())
		})
	})

})
