// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package executions

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"time"

	clientlib "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/landscaper/pkg/utils/read_write_layer"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	// . "github.com/onsi/gomega/gstruct"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	mockv1alpha1 "github.com/gardener/landscaper/apis/deployer/mock/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func GenerationHandlingTestsForNewReconcile(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "executions", "testdata", "test1")
	)

	Describe("Generation Handling", func() {

		var (
			state = f.Register()
			ctx   context.Context
		)

		log, err := logging.GetLogger()
		if err != nil {
			f.Log().Logfln("Error fetching logger: %w", err)
			return
		}

		BeforeEach(func() {
			ctx = context.Background()
			ctx = logging.NewContext(ctx, log)
		})

		AfterEach(func() {
			ctx.Done()
		})

		It("should correctly handle generations and observed generations for executions and their deployitems", func() {
			By("Create execution")
			exec := &lsv1alpha1.Execution{}
			utils.ExpectNoError(utils.ReadResourceFromFile(exec, path.Join(testdataDir, "00-execution.yaml")))
			exec.SetNamespace(state.Namespace)
			utils.ExpectNoError(state.Create(ctx, exec))

			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(utils.UpdateJobIdForExecutionC(ctx, f.Client, exec)).To(Succeed())

			By("verify that deployitem has been created")
			di := &lsv1alpha1.DeployItem{}
			Eventually(func() (bool, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)
				if err != nil {
					return false, err
				}

				deployItems, err := read_write_layer.ListManagedDeployItems(ctx, f.Client, clientlib.ObjectKeyFromObject(exec), read_write_layer.R000000)
				if err != nil || len(deployItems.Items) == 0 {
					return false, err
				}

				di = &deployItems.Items[0]

				return true, nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "unable to fetch deployitem")

			By("verify that execution is succeeded")
			Eventually(func() (lsv1alpha1.ExecutionPhase, error) {
				err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)
				if err != nil {
					return "", err
				}
				return exec.Status.ExecutionPhase, nil
			}, timeoutTime, resyncTime).Should(BeEquivalentTo(lsv1alpha1.ExecutionPhases.Succeeded), "execution should be in phase '%s'", lsv1alpha1.ExecutionPhases.Succeeded)

			oldExGen := exec.Generation
			oldDIGen := di.Generation
			Expect(exec.Status.ExecutionGenerations[0].ObservedGeneration).To(Equal(oldExGen))

			mockConfig := &mockv1alpha1.ProviderConfiguration{}
			utils.ExpectNoError(json.Unmarshal(di.Spec.Configuration.Raw, mockConfig))
			mockStatus := map[string]interface{}{}
			utils.ExpectNoError(json.Unmarshal(mockConfig.ProviderStatus.Raw, &mockStatus))
			Expect(mockStatus).To(HaveKeyWithValue("key", BeEquivalentTo("foo")))

			By("update execution")
			mockStatus["key"] = "bar"
			rawMockStatus, err := json.Marshal(mockStatus)
			utils.ExpectNoError(err)
			mockConfig.ProviderStatus = &runtime.RawExtension{
				Raw: rawMockStatus,
			}
			rawMockConfig, err := json.Marshal(mockConfig)
			utils.ExpectNoError(err)
			exec.Spec.DeployItems[0].Configuration = &runtime.RawExtension{
				Raw: rawMockConfig,
			}
			utils.ExpectNoError(state.Update(ctx, exec))

			Expect(state.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec)).To(Succeed())
			Expect(utils.UpdateJobIdForExecutionC(ctx, state.Client, exec)).To(Succeed())

			By("verify that deployitem has been changed")
			Eventually(func() (bool, error) {
				if err := f.Client.Get(ctx, kutil.ObjectKeyFromObject(di), di); err != nil {
					return false, err
				}
				mockConfig := &mockv1alpha1.ProviderConfiguration{}
				if err := json.Unmarshal(di.Spec.Configuration.Raw, mockConfig); err != nil {
					return false, err
				}
				mockStatus := map[string]interface{}{}
				if err := json.Unmarshal(mockConfig.ProviderStatus.Raw, &mockStatus); err != nil {
					return false, err
				}
				rawKey, ok := mockStatus["key"]
				if !ok {
					return false, fmt.Errorf("status does not contain 'key'")
				}
				key, ok := rawKey.(string)
				if !ok {
					return false, fmt.Errorf("value of key 'key' is not a string")
				}
				return key == "bar", nil
			}, timeoutTime, resyncTime).Should(BeTrue(), "deployitem should have been updated to match new execution spec")
			utils.ExpectNoError(f.Client.Get(ctx, kutil.ObjectKeyFromObject(exec), exec))

			By("verify that generations and observed generations behave as expected")
			Expect(exec.Generation).To(BeNumerically(">", oldExGen))
			Expect(di.Generation).To(BeNumerically(">", oldDIGen))
			Expect(exec.Status.ExecutionGenerations[0].ObservedGeneration).To(Equal(exec.Generation))

			By("delete execution")
			utils.ExpectNoError(utils.DeleteExecutionForNewReconcile(ctx, f.Client, exec, 2*time.Minute))
		})

	})
}
