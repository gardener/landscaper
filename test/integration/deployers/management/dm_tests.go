// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gardener/landscaper/pkg/utils/landscaper"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/onsi/ginkgo"
	g "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/landscaper/test/utils/envtest"

	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	"github.com/gardener/landscaper/apis/config"
	"github.com/gardener/landscaper/pkg/agent"
	"github.com/gardener/landscaper/pkg/api"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/deployer/mock"

	"github.com/gardener/landscaper/test/framework"
	testutil "github.com/gardener/landscaper/test/utils"
)

func DeployerManagementTests(f *framework.Framework) {
	ginkgo.Describe("Deployer Management", func() {
		var (
			state = f.Register()
			ctx   context.Context

			previousDeployerRegistrations sets.String
			previousEnvironments          sets.String
		)

		ginkgo.BeforeEach(func() {
			ctx = context.Background()
			drList := &lsv1alpha1.DeployerRegistrationList{}
			testutil.ExpectNoError(f.Client.List(ctx, drList))
			previousDeployerRegistrations = sets.NewString()
			for _, reg := range drList.Items {
				previousDeployerRegistrations.Insert(reg.Name)
			}

			envList := &lsv1alpha1.EnvironmentList{}
			testutil.ExpectNoError(f.Client.List(ctx, envList))
			previousEnvironments = sets.NewString()
			for _, env := range envList.Items {
				previousEnvironments.Insert(env.Name)
			}

		})

		ginkgo.AfterEach(func() {
			defer ctx.Done()
			drList := &lsv1alpha1.DeployerRegistrationList{}
			testutil.ExpectNoError(f.Client.List(ctx, drList))

			var allErrs []error
			for _, reg := range drList.Items {
				if previousDeployerRegistrations.Has(reg.Name) {
					continue
				}
				if err := envtest.CleanupForObject(ctx, f.Client, &reg, 2*time.Minute); err != nil {
					allErrs = append(allErrs, err)
				}
			}

			envList := &lsv1alpha1.EnvironmentList{}
			testutil.ExpectNoError(f.Client.List(ctx, envList))
			for _, env := range envList.Items {
				if previousEnvironments.Has(env.Name) {
					continue
				}
				if err := envtest.CleanupForObject(ctx, f.Client, &env, 2*time.Minute); err != nil {
					allErrs = append(allErrs, err)
				}
			}
			testutil.ExpectNoError(errors.NewAggregate(allErrs))

		})

		ginkgo.Context("Agent", func() {

			var (
				wg     sync.WaitGroup
				mgrCtx context.Context
				cancel context.CancelFunc
				mgr    manager.Manager

				numOfInstallations int
			)

			ginkgo.BeforeEach(func() {
				mgrCtx, cancel = context.WithCancel(context.Background())

				instList := &lsv1alpha1.InstallationList{}
				testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
				numOfInstallations = len(instList.Items)

				var err error
				mgr, err = manager.New(f.RestConfig, manager.Options{
					Scheme:             api.LandscaperScheme,
					MetricsBindAddress: "0",
				})
				testutil.ExpectNoError(err)

				err = agent.AddToManager(ctx, logging.Discard(), mgr, mgr, config.AgentConfiguration{
					Name:      "testenv",
					Namespace: state.Namespace,
				})
				testutil.ExpectNoError(err)

				wg = sync.WaitGroup{}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer ginkgo.GinkgoRecover()
					testutil.ExpectNoError(mgr.Start(mgrCtx))
				}()
			})

			ginkgo.AfterEach(func() {
				// cancel mgr context to close the manager watches.
				cancel()
				wg.Wait()

				// remove finalizer from testenv
				env := &lsv1alpha1.Environment{}
				env.Name = "testenv"
				testutil.ExpectNoError(envtest.CleanupForObject(ctx, f.Client, env, 5*time.Second))
			})

			ginkgo.It("should create and delete new installations for a new environment", func() {
				numOfDeployerRegistrations := previousDeployerRegistrations.Len()

				// the running agent should have created a new environment
				env := &lsv1alpha1.Environment{}
				envKey := kutil.ObjectKey("testenv", "")
				testutil.ExpectNoError(f.Client.Get(ctx, envKey, env))

				g.Eventually(func() error {
					instList := &lsv1alpha1.InstallationList{}
					testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
					newInstallations := len(instList.Items) - numOfInstallations
					if newInstallations != numOfDeployerRegistrations {
						err := fmt.Errorf("expect %d installation but found %d new", numOfDeployerRegistrations, newInstallations)
						f.TestLog().Logln(err.Error())
						return err
					}
					// expect that all installations are healthy
					var allErrs []error
					for _, inst := range instList.Items {
						finished, err := landscaper.IsInstallationFinished(&inst, lsv1alpha1.InstallationPhaseSucceeded)
						if err != nil {
							allErrs = append(allErrs, err)
						} else if !finished {
							allErrs = append(allErrs, fmt.Errorf("installation phase is not suceeded, but %s", inst.Status.InstallationPhase))
						}
					}
					if len(allErrs) != 0 {
						err := errors.NewAggregate(allErrs)
						f.TestLog().Logln(err.Error())
						return err
					}
					return nil
				}, 3*time.Minute, 10*time.Second).Should(g.Succeed())

				ginkgo.By("should delete the deployer when the Environment is removed")
				testutil.ExpectNoError(f.Client.Delete(ctx, env))

				g.Eventually(func() error {
					instList := &lsv1alpha1.InstallationList{}
					testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
					if len(instList.Items) != numOfInstallations {
						err := fmt.Errorf("expect %d installation but found %d", numOfDeployerRegistrations, len(instList.Items))
						f.TestLog().Logln(err.Error())
						return err
					}
					f.TestLog().Logfln("found %d installations", len(instList.Items))
					return nil
				}, 3*time.Minute, 10*time.Second).Should(g.Succeed())

				g.Eventually(func() error {
					if err := f.Client.Get(ctx, envKey, env); err != nil {
						return err
					}
					if len(env.Finalizers) != 1 {
						return fmt.Errorf("expected that the environment still has one finalizer but found %d", len(env.Finalizers))
					}
					return nil
				}, 30*time.Second, 1*time.Second).Should(g.Succeed())
			})
		})

		ginkgo.It("should manage a deployer's lifecycle for a new deployer registration", func() {
			instList := &lsv1alpha1.InstallationList{}
			testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
			numOfInstallations := len(instList.Items)
			previousInstallations := sets.String{}
			for _, inst := range instList.Items {
				previousInstallations.Insert(inst.Name)
			}
			numOfEnvironments := previousEnvironments.Len()

			repoCtx, err := cdv2.NewUnstructured(cdv2.NewOCIRegistryRepository("eu.gcr.io/gardener-project/development", ""))
			testutil.ExpectNoError(err)
			reg := &lsv1alpha1.DeployerRegistration{}
			reg.Name = "test-deployer"
			reg.Spec.DeployItemTypes = []lsv1alpha1.DeployItemType{mock.Type}
			reg.Spec.InstallationTemplate.ComponentDescriptor = &lsv1alpha1.ComponentDescriptorDefinition{
				Reference: &lsv1alpha1.ComponentDescriptorReference{
					RepositoryContext: &repoCtx,
					ComponentName:     "github.com/gardener/landscaper/mock-deployer",
					Version:           f.LsVersion,
				},
			}
			reg.Spec.InstallationTemplate.Blueprint.Reference = &lsv1alpha1.RemoteBlueprintReference{
				ResourceName: "mock-deployer-blueprint",
			}
			reg.Spec.InstallationTemplate.ImportDataMappings = map[string]lsv1alpha1.AnyJSON{
				"values": lsv1alpha1.NewAnyJSON([]byte("{}")),
			}

			testutil.ExpectNoError(state.Create(ctx, reg))

			g.Eventually(func() error {
				instList = &lsv1alpha1.InstallationList{}
				testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
				newInstallations := []lsv1alpha1.Installation{}
				for _, inst := range instList.Items {
					if !previousInstallations.Has(inst.Name) {
						newInstallations = append(newInstallations, inst)
					}
				}
				newInstallationCount := len(newInstallations)
				if newInstallationCount != numOfEnvironments {
					err := fmt.Errorf("expected %d new installations but found %d", numOfEnvironments, newInstallationCount)
					f.TestLog().Logln(err.Error())
					return err
				}
				// expect that all installations are healthy
				var allErrs []error
				for _, inst := range newInstallations {
					finished, err := landscaper.IsInstallationFinished(&inst, lsv1alpha1.InstallationPhaseSucceeded)
					if err != nil {
						allErrs = append(allErrs, err)
					} else if !finished {
						allErrs = append(allErrs, fmt.Errorf("installation phase is not suceeded, but %s", inst.Status.InstallationPhase))
					}
				}
				if len(allErrs) != 0 {
					err := errors.NewAggregate(allErrs)
					f.TestLog().Logln(err.Error())
					return err
				}
				return nil
			}, 3*time.Minute, 10*time.Second).Should(g.Succeed())

			ginkgo.By("should delete the deployer when the DeployerRegistration is removed")
			testutil.ExpectNoError(testutil.DeleteObject(ctx, f.Client, reg, 3*time.Minute))

			instList = &lsv1alpha1.InstallationList{}
			testutil.ExpectNoError(f.Client.List(ctx, instList, client.InNamespace(f.LsNamespace)))
			g.Expect(instList.Items).To(g.HaveLen(numOfInstallations), fmt.Sprintf("expected %d installations total but found %d", numOfInstallations, len(instList.Items)))
		})
	})
}
