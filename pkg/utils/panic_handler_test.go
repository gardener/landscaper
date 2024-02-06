// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lsutil "github.com/gardener/landscaper/pkg/utils"
)

type testController struct {
	innerReconcile func(ctx context.Context) (result reconcile.Result, err error)
}

func (c *testController) reconcile(ctx context.Context) (result reconcile.Result, err error) {
	result = reconcile.Result{}
	defer lsutil.HandlePanics(ctx, &result)

	result, err = c.innerReconcile(ctx)

	return result, err
}

var _ = Describe("Panic Handler", func() {

	It("should handle the case without a panic", func() {
		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {
				result := reconcile.Result{Requeue: true, RequeueAfter: 2 * time.Minute}
				return result, nil
			},
		}

		res, err := c.reconcile(context.Background())
		Expect(res.Requeue).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(2 * time.Minute))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle a nilpointer", func() {
		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {
				var n *int
				m := *n + 1 // provoke a nilpointer
				result := reconcile.Result{Requeue: true, RequeueAfter: time.Duration(m) * time.Minute}
				return result, nil
			},
		}

		res, err := c.reconcile(context.Background())
		Expect(res.Requeue).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(5 * time.Minute))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle index out of range", func() {
		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {
				names := []string{
					"a1",
					"a2",
					"a3",
				}
				fmt.Println("Name:", names[len(names)])
				result := reconcile.Result{Requeue: true, RequeueAfter: time.Duration(1) * time.Minute}
				return result, nil
			},
		}

		res, err := c.reconcile(context.Background())
		Expect(res.Requeue).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(5 * time.Minute))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle a panic other than a nilpointer", func() {
		defer func() {
			r := recover()
			Expect(r).NotTo(BeNil())
		}()

		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {
				if ctx != nil {
					panic("test")
				}
				result := reconcile.Result{Requeue: true, RequeueAfter: time.Minute}
				return result, nil
			},
		}

		_, err := c.reconcile(context.Background())
		Expect(err).To(BeNil())
	})

	It("should handle divide by zero", func() {
		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {

				a := 1
				b := 2
				b = a + b
				c := a / (4 - b - a)
				fmt.Println("C:", c)
				result := reconcile.Result{Requeue: true, RequeueAfter: time.Duration(1) * time.Minute}
				return result, nil
			},
		}

		res, err := c.reconcile(context.Background())
		Expect(res.Requeue).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(5 * time.Minute))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle type assertion error", func() {
		c := testController{
			innerReconcile: func(ctx context.Context) (reconcile.Result, error) {

				var i interface{} = "hello"

				_, ok := i.(int)
				if ok {
					fmt.Println("Successful type assertion")
				} else {
					fmt.Println("Type assertion failed")
				}

				j := i.(int) // This will cause a panic: interface conversion: interface {} is string, not int
				fmt.Println(j)

				result := reconcile.Result{Requeue: true, RequeueAfter: time.Duration(1) * time.Minute}
				return result, nil
			},
		}

		res, err := c.reconcile(context.Background())
		Expect(res.Requeue).To(BeTrue())
		Expect(res.RequeueAfter).To(Equal(5 * time.Minute))
		Expect(err).NotTo(HaveOccurred())
	})

})
