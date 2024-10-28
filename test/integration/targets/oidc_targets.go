package targets

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/core/v1alpha1/targettypes"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"
	"github.com/gardener/landscaper/test/framework"
	"github.com/gardener/landscaper/test/utils"
)

func OIDCTargetTests(ctx context.Context, f *framework.Framework) {

	Describe("OIDC Targets", func() {

		const (
			openIDConnectApiVersion = "authentication.gardener.cloud/v1alpha1"
			openIDConnectKind       = "OpenIDConnect"
		)

		var (
			testdataDir = filepath.Join(f.RootPath, "test", "integration", "testdata", "targets", "oidc-targets")
			state       = f.Register()
		)

		createOpenIDConnect := func(name, clientID, issuerURL, prefix string) (*unstructured.Unstructured, error) {
			unstr := &unstructured.Unstructured{}
			unstr.SetUnstructuredContent(map[string]interface{}{
				"spec": map[string]interface{}{
					"clientID":             clientID,
					"issuerURL":            issuerURL,
					"supportedSigningAlgs": []string{"RS256"},
					"usernameClaim":        "sub",
					"usernamePrefix":       prefix,
				},
			})
			unstr.SetAPIVersion(openIDConnectApiVersion)
			unstr.SetKind(openIDConnectKind)
			unstr.SetName(name)
			err := f.Client.Create(ctx, unstr)
			return unstr, err
		}

		deleteOpenIDConnect := func(name string) error {
			unstr := &unstructured.Unstructured{}
			unstr.SetAPIVersion(openIDConnectApiVersion)
			unstr.SetKind(openIDConnectKind)
			unstr.SetName(name)
			return f.Client.Delete(ctx, unstr)
		}

		createAdminClusterRoleBinding := func(name, saName, saNamespace, prefix string) error {
			b := &v12.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{},
				Subjects:   nil,
				RoleRef:    v12.RoleRef{},
			}
			b.SetName(name)
			b.RoleRef = v12.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}
			b.Subjects = []v12.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "User",
					Name:     fmt.Sprintf("%ssystem:serviceaccount:%s:%s", prefix, saNamespace, saName),
				},
			}

			return f.Client.Create(ctx, b)
		}

		createOIDCTarget := func(ctx context.Context, name, namespace, saName, saNamespace, audience string) (*lsv1alpha1.Target, error) {
			config := &targettypes.KubernetesClusterTargetConfig{
				OIDCConfig: &targettypes.OIDCConfig{
					Server: f.RestConfig.Host,
					CAData: f.RestConfig.CAData,
					ServiceAccount: v1.LocalObjectReference{
						Name: saName,
					},
					Audience:          []string{audience},
					ExpirationSeconds: ptr.To[int64](86400),
				},
			}
			configRaw, err := json.Marshal(config)
			if err != nil {
				return nil, err
			}

			t := &lsv1alpha1.Target{}
			t.SetName(name)
			t.SetNamespace(namespace)
			t.Spec = lsv1alpha1.TargetSpec{
				Type: targettypes.KubernetesClusterTargetType,
				Configuration: &lsv1alpha1.AnyJSON{
					RawMessage: configRaw,
				},
			}

			if err := state.Create(ctx, t); err != nil {
				return nil, err
			}
			return t, nil
		}

		It("should use an oidc target", func() {
			const (
				openIDConnectName  = "resource-cluster-oidc"
				targetName         = "my-cluster-oidc"
				serviceAccountName = "service-account-oidc"
				bindingName        = "binding-oidc"
				audience           = "target-cluster-oidc"
				prefix             = "resource-cluster-oidc:"
			)

			By("Create OpenIDConnect resource so that the target cluster trusts the resource cluster")
			_, err := createOpenIDConnect(openIDConnectName, audience, f.OIDCIssuerURL, prefix)
			Expect(err).NotTo(HaveOccurred())

			By("Create ClusterRoleBinding on target cluster for ServiceAccount on resource cluster")
			err = createAdminClusterRoleBinding(bindingName, serviceAccountName, state.Namespace, prefix)
			Expect(err).NotTo(HaveOccurred())

			By("Create ServiceAccount on resource cluster")
			_, err = utils.CreateServiceAccount(ctx, state.State, serviceAccountName, state.Namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Create oidc target on resource cluster")
			target, err := createOIDCTarget(ctx, targetName, state.Namespace, serviceAccountName, state.Namespace, audience)
			Expect(err).NotTo(HaveOccurred())

			By("Create DataObject for namespace import")
			doNamespace := &lsv1alpha1.DataObject{}
			utils.ExpectNoError(utils.CreateNamespaceDataObjectFromFile(ctx, state.State, doNamespace, path.Join(testdataDir, "import-do-namespace.yaml")))

			By("Create Installation")
			inst := &lsv1alpha1.Installation{}
			utils.ExpectNoError(utils.CreateInstallationFromFile(ctx, state.State, inst, path.Join(testdataDir, "installation.yaml")))

			By("Wait for Installation to finish")
			utils.ExpectNoError(lsutils.WaitForInstallationToFinish(ctx, f.Client, inst, lsv1alpha1.InstallationPhases.Succeeded, 2*time.Minute))

			By("Check deployed configmaps")
			cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm-1", Namespace: state.Namespace}}
			key := kutil.ObjectKeyFromObject(cm)
			Expect(f.Client.Get(ctx, key, cm)).To(Succeed())

			By("Delete installation")
			Expect(state.Client.Delete(ctx, inst)).To(Succeed())
			Expect(lsutils.WaitForInstallationToBeDeleted(ctx, f.Client, inst, 2*time.Minute)).To(Succeed())

			By("Delete DataObject")
			Expect(state.Client.Delete(ctx, doNamespace)).To(Succeed())

			By("Delete Target")
			Expect(state.Client.Delete(ctx, target)).To(Succeed())

			By("Delete ServiceAccount")
			Expect(utils.DeleteServiceAccount(ctx, state.State, serviceAccountName, state.Namespace)).To(Succeed())

			By("Delete OpenIDConnect resource")
			Expect(deleteOpenIDConnect(openIDConnectName)).To(Succeed())
		})

	})
}
