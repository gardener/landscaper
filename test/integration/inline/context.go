// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"context"
	"path"
	"path/filepath"
	"time"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func ContextTests(f *framework.Framework) {
	var (
		testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "inline")
	)

	Describe("Context", func() {

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

		It("a Blueprint should use a Context", func() {
			var (
				doNamespace = &lsv1alpha1.DataObject{} // contains the namespace of the ConfigMap to be deployed by the Installation
				contxt      = &lsv1alpha1.Context{}    // contains the repository context of the Installation
				inst        = &lsv1alpha1.Installation{}
			)

			By("Create DataObject with imports")
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "installation-context", "import-do-namespace.yaml")))

			By("Create Context")
			utils.ExpectNoError(utils.CreateContextFromFile(ctx, state.State, contxt, path.Join(testdataDir, "installation-context", "context.yaml")))

			By("Create Target")
			target, err := utils.BuildInternalKubernetesTarget(ctx, f.Client, state.Namespace, "my-cluster", f.RestConfig)
			utils.ExpectNoError(err)
			utils.ExpectNoError(state.Create(ctx, target))

			By("Create Installations with inline blueprint")
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation-context", "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed ConfigMap")
			expectedData := map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"}
			utils.ExpectNoError(utils.CheckConfigMap(ctx, state.State, "cm-example", expectedData))

			By("Delete Installations")
			utils.ExpectNoError(f.Client.Delete(ctx, inst))
			utils.ExpectNoError(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute))

			By("Check deletion of ConfigMap")
			configMapKey := client.ObjectKey{Namespace: state.Namespace, Name: "cm-example"}
			configMap := &k8sv1.ConfigMap{}
			err = f.Client.Get(ctx, configMapKey, configMap)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})
}
