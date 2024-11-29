package targets

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func OIDCTargetTests(ctx context.Context, f *framework.Framework) {

	Describe("OIDC Targets", func() {

		var (
			testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "targets", "oidc-targets")
			state       = f.Register()
		)

		It("should use an oidc target", func() {
			const (
				audience      = "oidc-target-cluster"
				configMapName = "oidc-target-test"
			)

			settings := map[string]any{
				"namespace":              state.Namespace,
				"openIDConnectName":      "landscaper-integration-test-oidc-targets",
				"clusterRoleBindingName": "landscaper:integration-test:oidc-targets",
				"serviceAccountName":     "oidc-serviceaccount",
				"targetName":             "oidc-target",
				"installationName":       "oidc-inst",
				"configMapName":          configMapName,
				"audience":               audience,
				"clientID":               audience,
				"issuerURL":              f.OIDCIssuerURL,
				"server":                 f.RestConfig.Host,
				"caData":                 base64.StdEncoding.EncodeToString(f.RestConfig.CAData),
				"prefix":                 "resource-cluster-oidc:",
			}

			By("Create OpenIDConnect resource so that the target cluster trusts the resource cluster")
			openIDConnect := &unstructured.Unstructured{}
			Expect(utils.CreateClientObjectFromTemplate(ctx, f.Client, filepath.Join(testdataDir, "openidconnect.yaml"), settings, openIDConnect)).To(Succeed())

			By("Create ClusterRoleBinding on resource cluster")
			clusterRoleBinding := &v12.ClusterRoleBinding{}
			Expect(utils.CreateClientObjectFromTemplate(ctx, f.Client, filepath.Join(testdataDir, "clusterrolebinding.yaml"), settings, clusterRoleBinding)).To(Succeed())

			By("Create ServiceAccount on resource cluster")
			serviceAccount := &v1.ServiceAccount{}
			Expect(utils.CreateStateObjectFromTemplate(ctx, state.State, filepath.Join(testdataDir, "serviceaccount.yaml"), settings, serviceAccount)).To(Succeed())

			By("Create OIDC Target on resource cluster")
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
			Expect(f.Client.Delete(ctx, clusterRoleBinding)).To(Succeed())

			By("Delete OpenIDConnect resource")
			Expect(f.Client.Delete(ctx, openIDConnect)).To(Succeed())
		})
	})
}
