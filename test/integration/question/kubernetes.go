// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package question

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/test/framework"
)

func KubernetesTests(f *framework.Framework) {

	Describe("Check kubernetes functions", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should check delete with wrong resource version", func() {
			cmName := "testcm"

			cm := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: state.Namespace},
				Immutable:  nil,
				Data:       map[string]string{"t1": "t1"},
			}

			By("Create cm")
			err := f.Client.Create(ctx, cm)
			Expect(err).To(Not(HaveOccurred()))

			By("Get cm")
			cmNew := &k8sv1.ConfigMap{}
			err = f.Client.Get(ctx, client.ObjectKeyFromObject(cm), cmNew)
			Expect(err).To(Not(HaveOccurred()))

			By("Update cm")
			cmNew.Data = map[string]string{"t1": "t2"}
			err = f.Client.Update(ctx, cmNew)
			Expect(err).To(Not(HaveOccurred()))

			By("Delete by outdated cm")
			err = f.Client.Delete(ctx, cm)
			Expect(err).To(HaveOccurred())

			By("Delete by last cm")
			err = f.Client.Delete(ctx, cmNew)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should check delete already deleted object", func() {
			cmName := "testcm"

			cm := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: state.Namespace},
				Immutable:  nil,
				Data:       map[string]string{"t1": "t1"},
			}

			By("Create cm")
			err := f.Client.Create(ctx, cm)
			Expect(err).To(Not(HaveOccurred()))

			By("Get cm")
			cmNew1 := &k8sv1.ConfigMap{}
			err = f.Client.Get(ctx, client.ObjectKeyFromObject(cm), cmNew1)
			Expect(err).To(Not(HaveOccurred()))

			cmNew2 := &k8sv1.ConfigMap{}
			err = f.Client.Get(ctx, client.ObjectKeyFromObject(cm), cmNew2)
			Expect(err).To(Not(HaveOccurred()))

			By("Delete first")
			err = f.Client.Delete(ctx, cmNew1)
			Expect(err).To(Not(HaveOccurred()))

			By("Delete second")
			err = f.Client.Delete(ctx, cmNew2)
			Expect(err).To(Not(HaveOccurred()))
		})
	})
}
