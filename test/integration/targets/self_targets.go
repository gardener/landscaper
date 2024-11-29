package targets

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func SelfTargetTests(ctx context.Context, f *framework.Framework) {

	Describe("Self Targets", func() {

		var (
			testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "targets", "self-targets")
			state       = f.Register()
		)

		It("should use a self target", func() {
			const (
				configMapName = "self-target-test"
			)

			settings := map[string]any{
				"namespace":              state.Namespace,
				"configMapName":          configMapName,
				"clusterRoleBindingName": "landscaper:integration-test:self-targets",
				"installationName":       "self-inst",
				"serviceAccountName":     "self-serviceaccount",
				"targetName":             "self-target",
			}

			By("Create ServiceAccount on resource cluster")
			serviceAccount := &v1.ServiceAccount{}
			Expect(utils.CreateStateObjectFromTemplate(ctx, state.State, filepath.Join(testdataDir, "serviceaccount.yaml"), settings, serviceAccount)).To(Succeed())

			By("Create ClusterRoleBinding on resource cluster")
			clusterRoleBinding := &v12.ClusterRoleBinding{}
			Expect(utils.CreateStateObjectFromTemplate(ctx, state.State, filepath.Join(testdataDir, "clusterrolebinding.yaml"), settings, clusterRoleBinding)).To(Succeed())

			By("Create Self Target")
			target := &lsv1alpha1.Target{}
			Expect(utils.CreateStateObjectFromTemplate(ctx, state.State, filepath.Join(testdataDir, "target.yaml"), settings, target)).To(Succeed())

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			Expect(utils.CreateStateObjectFromTemplate(ctx, state.State, filepath.Join(testdataDir, "installation.yaml"), settings, inst)).To(Succeed())

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed configmaps")
			cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: state.Namespace}}
			Expect(f.Client.Get(ctx, kutil.ObjectKeyFromObject(cm), cm)).To(Succeed())

			By("Delete installation")
			Expect(state.Client.Delete(ctx, inst)).To(Succeed())
			Expect(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute)).To(Succeed())

			By("Delete Target")
			Expect(state.Client.Delete(ctx, target)).To(Succeed())

			By("Delete ServiceAccount")
			Expect(state.Client.Delete(ctx, serviceAccount)).To(Succeed())

			By("Delete ClusterRoleBinding")
			Expect(state.Client.Delete(ctx, clusterRoleBinding)).To(Succeed())
		})
	})
}
